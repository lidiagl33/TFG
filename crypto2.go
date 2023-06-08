package main

import (
	"fmt"
	"math"

	"github.com/tuneinsight/lattigo/v3/ckks"
	"github.com/tuneinsight/lattigo/v3/dckks"
	"github.com/tuneinsight/lattigo/v3/drlwe"
	"github.com/tuneinsight/lattigo/v3/rlwe"
	"github.com/tuneinsight/lattigo/v3/utils"
)

type party struct {
	sk         *rlwe.SecretKey
	rlkEphemSk *rlwe.SecretKey

	ckgShare    *drlwe.CKGShare
	rkgShareOne *drlwe.RKGShare
	rkgShareTwo *drlwe.RKGShare
	rtgShare    *drlwe.RTGShare    // Rotation keys
	pcksShare   []*drlwe.PCKSShare // PKCS (public key switching) protocol

	input  [][]float64 // fingerprint
	NumRow int
	NumCol int
}

func getEncryptedPrediction(finalPrnu [][]PixelGray /*estimation user1: [rows][columns]*/, finalResiduals [][][]PixelGray /*[image][rows][columns]*/, N int /*numTestImages*/) []float64 {

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
	P := genparties(params, N)

	// Assign inputs (each residual to each user/party and the estimation of the prnu to the first one)
	getInputs(P, finalResiduals, finalPrnu)

	// 1) Collective public key generation
	pk := ckgphase(params, crs, P)

	// 2) Collective relinearization key generation
	rlk := rkgphase(params, crs, P) // sólo se necesita en el matching

	// 3) Collective rotation keys generation
	rtk := rtkphase(params, crs, P)

	// gets encrypted residuals
	encInputs := encPhase(params, P, pk, encoder)

	// Homomorphic additions of the ciphertexts to obtain the ENCODED PREDICTION
	encRes := evalPhase(params, encInputs, rlk, rtk) // matrix of ciphertexts

	// SIGUIENTE MEJORA -> usar extractLWEsample para optimizar (1) evita "innersum" (se extra el coeficiente de continua), y (2) el descifrado y la comunicación mejoran
	// key switching protocol -> encode over tpk
	encOut := pcksPhase(params, tpk, encRes, P) // array of ciphertexts -> ALL THE SAME

	// Decrypt the result with the target secret key
	fmt.Print("\n> Decrypt Phase\n")
	decryptor := ckks.NewDecryptor(params, tsk)

	// contains decrypted plaintext data
	ptres := make([]*ckks.Plaintext, len(encOut))
	for i := 0; i < len(ptres); i++ {
		ptres[i] = ckks.NewPlaintext(params, 1, params.DefaultScale())
	}

	for i := range encOut {
		decryptor.Decrypt(encOut[i], ptres[i])
	}

	// results of the encrypted prediction
	res := make([]float64, N) // len=N => SCORES (one per each residual)

	for i := 0; i < len(res); i++ { // len(res) = len(ptres) = N
		partialRes := encoder.Decode(ptres[i], params.LogSlots()) // len = 2^11 = 2048 (all values are practically equal)
		// we only need one because they are all the same
		res[i] = real(partialRes[0])
	}

	fmt.Print("\n> Finish Encryption\n\n")

	return res
}

func genparties(params ckks.Parameters, N int) []*party {

	P := make([]*party, N+1) // numParties = N+1 = numTestImg + 1(estimation of the prnu)
	for i := range P {
		pi := &party{}
		// Generates the invidividual secret key and for each Forensic Party P[i]
		pi.sk = ckks.NewKeyGenerator(params).GenSecretKey()
		P[i] = pi
		//P[i].sk = ckks.NewKeyGenerator(params).GenSecretKey()
	}

	return P
}

func getInputs(p []*party, residuals [][][]PixelGray, finalPrnus [][]PixelGray) {

	// we suppose only 1 ESTIMATION of the prnu (1 user)

	// residuals => [images][rows][columns]
	for i := 0; i < len(p); i++ { // len(P) = len(residuals) + 1 (estimation)

		in := make([][]float64, len(residuals[0])) // len(residuals[0]) = len(finalPrnus[0]) (lengthX)

		for t := 0; t < len(in); t++ {
			in[t] = make([]float64, len(residuals[0][0])) // len(residuals[0][0]) = len(finalPrnus[0][0]) (lenghtY)
		}

		p[i].input = in

		// the 1st party will keep the estimation of the prnu, the others will keep the residuals
		if i == 0 {
			for j := 0; j < len(finalPrnus); j++ {
				for k := 0; k < len(finalPrnus[0]); k++ {
					// asigns the estimation to the first party
					p[i].input[j][k] = finalPrnus[j][k].pix
				}
			}
		} else {
			for j := 0; j < len(residuals[i-1]); j++ {
				for k := 0; k < len(residuals[i-1][j]); k++ {
					// asigns each residual to its correspondant party
					p[i].input[j][k] = residuals[i-1][j][k].pix
				}
			}
		}

		p[i].NumRow = len(p[i].input)
		p[i].NumCol = len(p[i].input[0])

	}

}

func ckgphase(params ckks.Parameters, crs utils.PRNG, P []*party) *rlwe.PublicKey {

	ckg := dckks.NewCKGProtocol(params) // Public key generation
	ckgCombined := ckg.AllocateShare()

	for _, pi := range P {
		pi.ckgShare = ckg.AllocateShare()
	}

	crp := ckg.SampleCRP(crs)

	for _, pi := range P {
		ckg.GenShare(pi.sk, crp, pi.ckgShare)
	}

	pk := ckks.NewPublicKey(params)

	for _, pi := range P {
		ckg.AggregateShare(pi.ckgShare, ckgCombined, ckgCombined)
	}

	ckg.GenPublicKey(ckgCombined, crp, pk)

	return pk
}

func rkgphase(params ckks.Parameters, crs utils.PRNG, P []*party) *rlwe.RelinearizationKey {

	rkg := dckks.NewRKGProtocol(params) // Relinearization key generation
	_, rkgCombined1, rkgCombined2 := rkg.AllocateShare()

	for _, pi := range P {
		pi.rlkEphemSk, pi.rkgShareOne, pi.rkgShareTwo = rkg.AllocateShare()
	}

	crp := rkg.SampleCRP(crs)

	for _, pi := range P {
		rkg.GenShareRoundOne(pi.sk, crp, pi.rlkEphemSk, pi.rkgShareOne)
	}

	for _, pi := range P {
		rkg.AggregateShare(pi.rkgShareOne, rkgCombined1, rkgCombined1)
	}

	for _, pi := range P {
		rkg.GenShareRoundTwo(pi.rlkEphemSk, pi.sk, rkgCombined1, pi.rkgShareTwo)
	}

	rlk := ckks.NewRelinearizationKey(params)

	for _, pi := range P {
		rkg.AggregateShare(pi.rkgShareTwo, rkgCombined2, rkgCombined2)
	}

	rkg.GenRelinearizationKey(rkgCombined1, rkgCombined2, rlk)

	return rlk
}

func rtkphase(params ckks.Parameters, crs utils.PRNG, P []*party) *rlwe.RotationKeySet {

	rtg := dckks.NewRotKGProtocol(params) // Rotation keys generation

	for _, pi := range P {
		pi.rtgShare = rtg.AllocateShare()
	}

	galEls := params.GaloisElementsForRowInnerSum()
	rotKeySet := ckks.NewRotationKeySet(params, galEls)

	for _, galEl := range galEls {

		rtgShareCombined := rtg.AllocateShare()

		crp := rtg.SampleCRP(crs)

		for _, pi := range P {
			rtg.GenShare(pi.sk, galEl, crp, pi.rtgShare)
		}

		for _, pi := range P {
			rtg.AggregateShare(pi.rtgShare, rtgShareCombined, rtgShareCombined)
		}
		rtg.GenRotationKey(rtgShareCombined, crp, rotKeySet.Keys[galEl])
	}

	return rotKeySet
}

func encPhase(params ckks.Parameters, P []*party, pk *rlwe.PublicKey, encoder ckks.Encoder) (encInputs [][][]*ckks.Ciphertext) {

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

	// create cyphertexts -> encrypt residuals and estimation
	for i, pi := range P {
		for j := 0; j < NumRowEncIn; j++ {
			for k := 0; k < NumColEncIn; k++ {

				// rellenar con ceros el ciphertext (si es más grande que los elementos que quedan por cifrar)
				if (k+1)*params.Slots() > len(pi.input[j]) {

					zeros := (k+1)*params.Slots() - len(pi.input[j]) // number of zeros needed
					//fmt.Printf("Number of 0s needed to fill the ciphertext: %d\n", zeros)

					add := make([]float64, zeros) // slice of 0s
					pi.input[j] = append(pi.input[j], add...)
				}

				//fmt.Printf("SIZE EACH ROW: %d\n", len(pi.input[j][(k*params.Slots()):((k+1)*params.Slots()-1)]))
				//fmt.Printf("values are %d y %d\n", k*params.Slots(), (k+1)*params.Slots())
				//fmt.Printf("size total row %d\n", len(pi.input[j]))

				// returns the data in a Plaintext, now it can pass it to the function Encrypt
				encoder.Encode(pi.input[j][(k*params.Slots()):((k+1)*params.Slots())], pt, params.LogSlots()) // go indexes [0:n] as the values 0, 1, ..., n - 1
				// encrypts the plaintex "pt" into a ciphertext
				encryptor.Encrypt(pt, encInputs[i][j][k])
			}
		}
	}

	return
}

func evalPhase(params ckks.Parameters, encInputs [][][]*ckks.Ciphertext, rlk *rlwe.RelinearizationKey, rtk *rlwe.RotationKeySet) (encRes []*ckks.Ciphertext) {

	// Rows and Columns for the "matrices of ciphertexts"
	NumRowEncIn := len(encInputs[0])
	NumColEncIn := len(encInputs[0][0])

	// encRes array de ciphertexts ([])
	encRes = make([]*ckks.Ciphertext, len(encInputs)-1)
	for i := 0; i < len(encRes); i++ {
		encRes[i] = ckks.NewCiphertext(params, 1, 1, params.DefaultScale())
	}

	// to save the results of the multiplications
	resMult := make([][][]*ckks.Ciphertext, len(encInputs)-1) // [parties][rows][columns] (row ciphertext) -> numParties-1 = numResiduals
	for i := range resMult {
		resMult[i] = make([][]*ckks.Ciphertext, NumRowEncIn)
		for j := range resMult[i] {
			resMult[i][j] = make([]*ckks.Ciphertext, NumColEncIn)
		}
	}
	// Initializes "input" ciphertexts
	for i := range resMult {
		for j := range resMult[i] {
			for k := range resMult[i][j] {
				resMult[i][j][k] = ckks.NewCiphertext(params, 1, 1, params.DefaultScale())
			}
		}
	}

	// to save the results of the total adding
	resAdd := make([]*ckks.Ciphertext, len(encInputs)-1) // array de ciphertext de len=N
	for i := range resAdd {
		resAdd[i] = ckks.NewCiphertext(params, 2, 1, params.DefaultScale()) // degree 2
	}

	// used after to make the different operations between the ciphertexts
	evaluator := ckks.NewEvaluator(params, rlwe.EvaluationKey{Rlk: rlk, Rtks: rtk}) // if using evaluator.innersum, we have to generate the power-of-two rotations

	for i := 1; i < len(encInputs); i++ { // party (begining from the second) => encInputs[0] = estimation of the prnu
		for j := 0; j < len(encInputs[0]); j++ { // NumRowEncIn
			for k := 0; k < len(encInputs[0][0]); k++ { // NumColEncIn
				//evaluator.Add(encRes[j][k], encInputs[i][j][k], encRes[j][k]) => encoded aggregation

				// 1) Multiplication of the fingerprint "query" / estimation (1st party) with the residual of each image (rest of parties)
				evaluator.Mul(encInputs[0][j][k], encInputs[i][j][k], resMult[i-1][j][k])
			}
		}
	}

	for i := 0; i < len(resMult); i++ {
		for j := 0; j < len(resMult[0]); j++ {
			for k := 0; k < len(resMult[0][0]); k++ {
				// 2) Addition of all the ciphertexts (BE CAREFUL WITH THE DEGREES)
				evaluator.Add(resAdd[i], resMult[i][j][k], resAdd[i])
			}
		}
		// 3) Relinearization
		evaluator.Relinearize(resAdd[i], resAdd[i]) // solo hay 1 ciphertext en cada party
	}

	for i := 0; i < len(encRes); i++ { // len(encRes) = len (resAdd)
		// 4) InnerSum
		evaluator.InnerSumLog(resAdd[i], 1, params.Slots(), encRes[i]) //encRes[i]->encRes[i] ( // batch, n = num of rotations ??)
	}

	return
}

func pcksPhase(params ckks.Parameters, tpk *rlwe.PublicKey, encRes []*ckks.Ciphertext, P []*party) (encOut []*ckks.Ciphertext) {

	// Collective key switching protocol from the collective secret key to the target public key
	// cambio de la global secret key a la "target secret key"

	// CHECK -> encOut and encRes are matrices of ciphertexts now
	pcks := dckks.NewPCKSProtocol(params, 3.19)

	for _, pi := range P {
		pi.pcksShare = make([]*drlwe.PCKSShare, len(encRes))
		for i := range encRes {
			pi.pcksShare[i] = pcks.AllocateShare(encRes[0].Level())
		}
	}

	for _, pi := range P {
		for i := range encRes {
			pcks.GenShare(pi.sk, tpk, encRes[i].Value[1], pi.pcksShare[i])
		}
	}

	pcksCombined := make([]*drlwe.PCKSShare, len(encRes))
	encOut = make([]*ckks.Ciphertext, len(encRes))
	for i := range encRes {
		pcksCombined[i] = pcks.AllocateShare(encRes[0].Level())
		encOut[i] = ckks.NewCiphertext(params, 1, 1, params.DefaultScale())
	}

	for _, pi := range P {
		for i := range encRes {
			pcks.AggregateShare(pi.pcksShare[i], pcksCombined[i], pcksCombined[i])
		}
	}

	for i := range encRes {
		pcks.KeySwitch(encRes[i], pcksCombined[i], encOut[i]) // switching the key
	}

	return

}
