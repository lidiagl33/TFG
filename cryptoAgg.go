package main

import (
	"fmt"
	"math"
	"time"

	"github.com/tuneinsight/lattigo/v3/ckks"
	"github.com/tuneinsight/lattigo/v3/dckks"
	"github.com/tuneinsight/lattigo/v3/drlwe"
	"github.com/tuneinsight/lattigo/v3/rlwe"
	"github.com/tuneinsight/lattigo/v3/utils"
)

type partyAg struct {
	sk        *rlwe.SecretKey
	ckgShare  *drlwe.CKGShare
	pcksShare [][]*drlwe.PCKSShare // PKCS (public key switching) protocol
	input     [][]float64          // fingerprint
	NumRow    int
	NumCol    int
}

var elapsedEncryptPartyAg time.Duration
var elapsedEncryptCloudAg time.Duration
var elapsedCKGCloudAg time.Duration
var elapsedCKGPartyAg time.Duration
var elapsedPCKSCloudAg time.Duration
var elapsedPCKSPartyAg time.Duration
var elapsedEvalCloudCPUAg time.Duration
var elapsedEvalPartyAg time.Duration
var elapsedDecCloudAg time.Duration
var elapsedDecPartyAg time.Duration

func getEncryptedAggregation(prnus [][][]PixelGray, N int, weights []float64) ([][]float64, [][]float64) {

	// VALUES IN A CIPHERTEXT:
	// with CKKS -> 2^LogSlots (here LogSlots = 11 => 2^11 = 2048 values)
	// with BFV -> 2^logN

	// ENCRYPTION PARAMETERS

	paramsDef := ckks.PN12QP109
	// Creating encryption parameters from a default param
	params, err := ckks.NewParametersFromLiteral(paramsDef)
	if err != nil {
		panic(err)
	}

	// crs = 'commom random string'
	// create the "password" to make the protocols work (common to all the parts)
	crs, err := utils.NewKeyedPRNG([]byte{'f', 'e', 'l', 'd', 's', 'p', 'a', 'r'}) //'t', 'r', 'u', 'm', 'p', 'e', 't'
	if err != nil {
		panic(err)
	}

	// "preparing" everything to encrypt
	encoder := ckks.NewEncoder(params)

	// Target private and public keys
	// the person who has the "tsk" could decode the content in the end
	// the "tpk" is the key which the information will be encoded with (after a "change of public key")
	tsk, tpk := ckks.NewKeyGenerator(params).GenKeyPair()

	// Create each party and allocate the memory for all the shares that the protocols will need
	P := genpartiesAg(params, N)

	// Assign inputs (each prnu to each user/party)
	expRes := getInputsAg(P, prnus, weights)

	// Collective public key generation
	pk := ckgphaseAg(params, crs, P)

	fmt.Printf("\tSETUP done (cloud: %s, party: %s)\n",
		elapsedCKGCloudAg, elapsedCKGPartyAg)

	// gets encrypted prnus
	encInputs := encPhaseAg(params, P, pk, encoder)

	elapsedEncryptCloudAg = elapsedEncryptPartyAg * time.Duration(len(P))
	fmt.Printf("\tENCRYPTION done (cloud: %s, party: %s)\n",
		elapsedEncryptCloudAg, elapsedEncryptPartyAg)

	// Homomorphic additions of the ciphertexts to obtain the ENCODED AGGREGATION
	encRes := evalPhaseAg(params, encInputs) // matrix of ciphertexts

	elapsedEvalPartyAg = elapsedEvalCloudCPUAg / time.Duration(len(P))
	fmt.Printf("\tEVALUATION done (cloud: %s, party: %s)\n",
		elapsedEvalCloudCPUAg, elapsedEvalPartyAg)

	// key switching protocol -> encode over tpk
	encOut := pcksPhaseAg(params, tpk, encRes, P) // matrix of ciphertexts

	fmt.Printf("\tKEY SWITCHING done (cloud: %s, party: %s)\n",
		elapsedPCKSCloudAg, elapsedPCKSPartyAg)

	// Decrypt the result with the target secret key
	fmt.Print("\n> Decrypt Phase\n")
	decryptor := ckks.NewDecryptor(params, tsk)

	// contains decrypted plaintext data
	ptres := make([][]*ckks.Plaintext, len(encOut))
	for i := range encOut {
		ptres[i] = make([]*ckks.Plaintext, len(encOut[i]))
		for j := range encOut[i] {
			ptres[i][j] = ckks.NewPlaintext(params, 1, params.DefaultScale())
		}
	}

	elapsedDecPartyAg = runTimed(func() {
		for i := range encOut {
			for j := range encOut[i] {
				decryptor.Decrypt(encOut[i][j], ptres[i][j])
			}
		}
	})

	elapsedDecCloudAg = elapsedDecPartyAg * time.Duration(len(P))
	fmt.Printf("\tDECRYPTION done (cloud: %s, party: %s)\n", elapsedDecCloudAg, elapsedDecPartyAg)

	// Check the result
	res := make([][]float64, P[0].NumRow)
	for i := range ptres {
		res[i] = make([]float64, P[0].NumCol)
	}

	for i := range ptres {
		for j := range ptres[i] {
			// decode the data into float64 data, with the same sizes of the initial prnus
			partialRes := encoder.Decode(ptres[i][j], params.LogSlots())
			for k := 0; k < len(res[0]); k++ {
				res[i][(j*len(partialRes) + k)] = real(partialRes[k])
			}
		}
	}

	for i := 0; i < len(res); i++ {
		for j := 0; j < len(res[0]); j++ {
			res[i][j] = res[i][j] / float64(N)
		}
	}

	var tolerancia = 1e-4
	for i := range expRes {
		for j := range expRes[i] {
			if expRes[i][j] != res[i][j] {
				delta := math.Abs(expRes[i][j] - res[i][j])
				if delta < tolerancia {
					// OK
				} else {
					fmt.Printf("\tincorrect\n error in position [%d][%d]\n", i, j)
				}
			}
		}
	}

	fmt.Printf("\tTOTAL TIME: %s\n", elapsedCKGCloudAg+elapsedEncryptCloudAg+
		elapsedEvalCloudCPUAg+elapsedPCKSCloudAg+elapsedDecCloudAg)

	fmt.Print("\n> Finish Encryption\n\n")

	return res, expRes
}

func genpartiesAg(params ckks.Parameters, N int) []*partyAg {

	P := make([]*partyAg, N)
	for i := range P {
		pi := &partyAg{}
		// Generates the invidividual secret key and for each Forensic Party P[i]
		pi.sk = ckks.NewKeyGenerator(params).GenSecretKey()
		P[i] = pi
		//P[i].sk = ckks.NewKeyGenerator(params).GenSecretKey()
	}

	return P
}

func getInputsAg(p []*partyAg, prnus [][][]PixelGray, weights []float64) (expRes [][]float64) {

	in := make([][]float64, len(prnus[0]))

	for t := 0; t < len(in); t++ {
		in[t] = make([]float64, len(prnus[0][0]))
	}

	// prnus => [user][rows][columns]
	for i := 0; i < len(p); i++ { // len(P) = len(prnus)

		p[i].input = in

		for j := 0; j < len(prnus[i]); j++ {
			for k := 0; k < len(prnus[i][j]); k++ {
				// asigns each prnu to its correspondant party
				p[i].input[j][k] = prnus[i][j][k].pix * weights[i]
			}
		}

		p[i].NumRow = len(p[i].input)
		p[i].NumCol = len(p[i].input[0])

	}

	// RESULTADOS AGREGACIIÓN EN CLARO
	expRes = make([][]float64, p[0].NumRow)
	for i := range expRes {
		expRes[i] = make([]float64, p[0].NumCol)
	}

	// Generate Aggregation Expected Results
	for _, pi := range p {
		for i := range pi.input {
			for j := range pi.input[i] {
				expRes[i][j] += pi.input[i][j] // pesos ya usados antes, (* weights[k])
			}
		}
	}

	for i := 0; i < len(expRes); i++ {
		for j := 0; j < len(expRes[0]); j++ {
			expRes[i][j] = expRes[i][j] / float64(len(p))
		}
	}

	return expRes

}

func ckgphaseAg(params ckks.Parameters, crs utils.PRNG, P []*partyAg) *rlwe.PublicKey {

	ckg := dckks.NewCKGProtocol(params) // Public key generation
	ckgCombined := ckg.AllocateShare()

	for _, pi := range P {
		pi.ckgShare = ckg.AllocateShare()
	}

	crp := ckg.SampleCRP(crs)

	elapsedCKGPartyAg = runTimedParty(func() {
		for _, pi := range P {
			ckg.GenShare(pi.sk, crp, pi.ckgShare)
		}
	}, len(P))

	pk := ckks.NewPublicKey(params)

	elapsedCKGCloudAg = runTimed(func() {
		for _, pi := range P {
			ckg.AggregateShare(pi.ckgShare, ckgCombined, ckgCombined)
		}
		ckg.GenPublicKey(ckgCombined, crp, pk)
	})

	fmt.Printf("\tckgphase done (cloud: %s, party: %s)\n", elapsedCKGCloudAg, elapsedCKGPartyAg)

	return pk
}

func encPhaseAg(params ckks.Parameters, P []*partyAg, pk *rlwe.PublicKey, encoder ckks.Encoder) (encInputs [][][]*ckks.Ciphertext) {

	NumRowEncIn := P[0].NumRow
	NumColEncIn := int(math.Ceil(float64(P[0].NumCol) / float64(params.Slots()))) // ceil => round up

	// SIZE OF THE CIPHERTEXT: 2048 values => maybe it will be necessary to complete with 0s

	encInputs = make([][][]*ckks.Ciphertext, len(P)) // [parties][rows][columns]
	for i := range encInputs {
		encInputs[i] = make([][]*ckks.Ciphertext, NumRowEncIn)
		for j := range encInputs[i] {
			encInputs[i][j] = make([]*ckks.Ciphertext, NumColEncIn)
		}
	}

	//encOut[i] = ckks.NewCiphertext(params, encRes[0].Degree(), encRes[0].Level(), encRes[0].Scale)

	// Initializes "input" ciphertexts
	for i := range encInputs {
		for j := range encInputs[i] {
			for k := range encInputs[i][j] {
				encInputs[i][j][k] = ckks.NewCiphertext(params, 1 /*int(params.N())*/, 1, params.DefaultScale())
			}
		}
	}

	// Each party encrypts its bidimensional array of input vectors into a bidimensional array of input ciphertexts
	fmt.Print("\n> Encrypt Phase\n")
	encryptor := ckks.NewEncryptor(params, pk)

	pt := ckks.NewPlaintext(params, 1, params.DefaultScale())

	// create cyphertexts
	elapsedEncryptPartyAg = runTimedParty(func() {
		for i, pi := range P {
			for j := 0; j < NumRowEncIn; j++ {
				for k := 0; k < NumColEncIn; k++ {

					//rellenar con ceros el ciphertext (si es más grande que los elementos que quedan por cifrar)
					if (k+1)*params.Slots() > len(pi.input[j]) {

						zeros := (k+1)*params.Slots() - len(pi.input[j]) // number of zeros needed
						//fmt.Printf("Number of 0s needed to fill the ciphertext: %d\n", zeros)

						add := make([]float64, zeros) // slice of 0s
						pi.input[j] = append(pi.input[j], add...)
					}

					fmt.Printf("SIZE EACH ROW: %d\n", len(pi.input[j][(k*params.Slots()):((k+1)*params.Slots()-1)]))
					fmt.Printf("values are %d y %d\n", k*params.Slots(), (k+1)*params.Slots())
					fmt.Printf("size total row %d\n", len(pi.input[j]))

					// returns the data in a Plaintext, now it can pass it to the function Encrypt
					encoder.Encode(pi.input[j][(k*params.Slots()):((k+1)*params.Slots())], pt, params.LogSlots()) // go indexes [0:n] as the values 0, 1, ..., n - 1
					// encrypts the plaintex "pt" into a ciphertext
					encryptor.Encrypt(pt, encInputs[i][j][k])
				}
			}
		}
	}, len(P))

	elapsedEncryptCloudAg = time.Duration(0)
	fmt.Printf("\tencPhase done (cloud: %s, party: %s)\n", elapsedEncryptCloudAg, elapsedEncryptPartyAg)

	return
}

func evalPhaseAg(params ckks.Parameters, encInputs [][][]*ckks.Ciphertext) (encRes [][]*ckks.Ciphertext) {

	// Rows and Columns for the "matrices of ciphertexts"
	NumRowEncIn := len(encInputs[0])
	NumColEncIn := len(encInputs[0][0])

	encRes = make([][]*ckks.Ciphertext, NumRowEncIn)
	for i := 0; i < len(encRes); i++ {
		encRes[i] = make([]*ckks.Ciphertext, NumColEncIn)
	}

	for i := 0; i < len(encRes); i++ {
		for j := 0; j < len(encRes[0]); j++ {
			encRes[i][j] = ckks.NewCiphertext(params, 1, 1, params.DefaultScale())
		}
	}

	// used after to make the addition between the ciphertexts
	evaluator := ckks.NewEvaluator(params, rlwe.EvaluationKey{Rlk: nil, Rtks: nil})

	elapsedEvalCloudCPUAg = runTimed(func() {
		for i := 0; i < len(encInputs); i++ {
			for j := 0; j < len(encInputs[0]); j++ { // NumRowEncIn
				for k := 0; k < len(encInputs[0][0]); k++ { // NumColEncIn
					evaluator.Add(encRes[j][k], encInputs[i][j][k], encRes[j][k])
				}
			}
		}
	})

	elapsedEvalPartyAg = time.Duration(0)
	fmt.Printf("\tevalPhase done (cloud: %s, party: %s)\n", elapsedEvalCloudCPUAg, elapsedEvalPartyAg)

	return
}

func pcksPhaseAg(params ckks.Parameters, tpk *rlwe.PublicKey, encRes [][]*ckks.Ciphertext, P []*partyAg) (encOut [][]*ckks.Ciphertext) {

	// Collective key switching protocol from the collective secret key to the target public key

	pcks := dckks.NewPCKSProtocol(params, 3.19)

	for _, pi := range P {
		pi.pcksShare = make([][]*drlwe.PCKSShare, len(encRes))
		for i := range encRes {
			pi.pcksShare[i] = make([]*drlwe.PCKSShare, len(encRes[i]))
			for j := range encRes[0] {
				pi.pcksShare[i][j] = pcks.AllocateShare(encRes[0][0].Level())
			}
		}
	}

	elapsedPCKSPartyAg = runTimedParty(func() {
		for _, pi := range P {
			for i := range encRes {
				for j := range encRes[0] {
					pcks.GenShare(pi.sk, tpk, encRes[i][j].Value[1], pi.pcksShare[i][j])
				}
			}
		}
	}, len(P))

	pcksCombined := make([][]*drlwe.PCKSShare, len(encRes))
	encOut = make([][]*ckks.Ciphertext, len(encRes))
	for i := range encRes {
		pcksCombined[i] = make([]*drlwe.PCKSShare, len(encRes[i]))
		encOut[i] = make([]*ckks.Ciphertext, len(encRes[i]))
		for j := range encRes[0] {
			pcksCombined[i][j] = pcks.AllocateShare(encRes[0][0].Level())
			encOut[i][j] = ckks.NewCiphertext(params, 1, 1, params.DefaultScale())
		}
	}

	elapsedPCKSCloudAg = runTimed(func() {
		for _, pi := range P {
			for i := range encRes {
				for j := range encRes[0] {
					pcks.AggregateShare(pi.pcksShare[i][j], pcksCombined[i][j], pcksCombined[i][j])
				}
			}
		}
		for i := range encRes {
			for j := range encRes[0] {
				pcks.KeySwitch(encRes[i][j], pcksCombined[i][j], encOut[i][j]) // switching the key
			}
		}
	})

	fmt.Printf("\tpcksphase done (cloud: %s, party: %s)\n", elapsedPCKSCloudAg, elapsedPCKSPartyAg)

	return

}
