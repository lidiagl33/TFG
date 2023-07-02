package main

import (
	"C"
	"fmt"
	"strconv"

	"gocv.io/x/gocv"
)

// RGBA pixel
type Pixel struct {
	R float64
	G float64
	B float64
	A float64
}

// Gray pixel
type PixelGray struct {
	pix float64
}

func main() {

	fmt.Print("\n\n#############\n")
	fmt.Println("    BEGIN")
	fmt.Print("#############\n\n\n")

	// ########## SETUP PARAMETERS ##########

	const LENGTHIMGX int = 250
	const LENGTHIMGY int = 250

	var numCam int = 4
	var numExp int = 1 // experiment
	var numParties int = 1
	//var imgPerCam int = 70
	//var imgExtraction int = 40
	//var imgTest int = 30

	// ########## POST-SETUP ##########

	var data = make(map[string][]gocv.Mat) // name of the user : array of images
	var numUsers int
	var nameUsers []string

	data, numUsers, nameUsers = getDataExp(numParties, numExp, LENGTHIMGX, LENGTHIMGY) // read the images

	// PRUEBAS PRODUCTO ESCALAR RESIDUOS => ORIENTACION IMAGENES
	/*
		// ############################################################
		//gocv.Rotate(data[nameUsers[0]][1], &data[nameUsers[0]][1], gocv.Rotate180Clockwise) // rotation of the 2nd image

		aux := getResiduals(data[nameUsers[0]], "", LENGTHIMGX, LENGTHIMGY) // [layer B/G/R][image][rows prnu][columns prnu]

		resfin1 := calculateFinalResidual(aux[0][0], aux[1][0], aux[2][0]) // 1st image
		resfin2 := calculateFinalResidual(aux[0][1], aux[1][1], aux[2][1]) // 2nd image

		//gocv.Rotate(resfin2, &resfin2, gocv.Rotate90Clockwise)
		result := scalarProduct(resfin1, resfin2)

		fmt.Print(result)
		// ############################################################
	*/

	// ########## LOCAL EXTRACTION ##########

	var PRNUS = make(map[string][][][]PixelGray) // [layer B/G/R][rows prnu][columns prnu]

	for i := 0; i < numUsers; i++ {
		// does it one time per user
		// if the last parameter is "true" => the function will check the results
		PRNUS[nameUsers[i]] = extraction(data[nameUsers[i]], nameUsers[i], LENGTHIMGX, LENGTHIMGY, false)
	}

	fmt.Println("EXTRACTION DONE")

	var prnusB, prnusG, prnusR [][][]PixelGray // [user][rows][columns]
	var finalPrnus [][][]PixelGray             // [user][rows][columns] -> calculated by 0.3*K_R+0.59*K_G+0.11*K_B

	for i := 0; i < numUsers; i++ {
		prnusUser := PRNUS[nameUsers[i]] // PRNUS B, G, R (each one is an matrix[rows][columns])
		prnusB = append(prnusB, prnusUser[0])
		prnusG = append(prnusG, prnusUser[1])
		prnusR = append(prnusR, prnusUser[2])
		finalPrnus = append(finalPrnus, calculateFinalK(prnusUser[0], prnusUser[1], prnusUser[2]))
	}

	var variance []float64 // [variance of the K estimation], len(variance)=numParties
	var weights []float64
	var optimalWeights bool = false

	if numParties != 1 {
		if optimalWeights {
			variance = calculateVariance(finalPrnus, numParties)
			writeLines(variance, "./experiment"+strconv.Itoa(numExp)+"/variances.txt")
			//time.Sleep(20 * time.Second) => go to MATLAB to calculate the weights after reading the variances
			weights = readLines("./experiment" + strconv.Itoa(numExp) + "/weights.txt")
		} else {
			// weights with numParties values, each one being 1/numParties
			for i := 0; i < numParties; i++ {
				weights = append(weights, float64(1/float64(numParties)))
			}
		}
	} else {
		weights = append(weights, 1)
	}

	// ########## AGGREGATION WITHOUT ENCODING ##########

	var agreg [][]PixelGray

	agreg = getAgregation(finalPrnus, numUsers, weights)
	fmt.Print(agreg[0][0])

	fmt.Println("AGGREGATION DONE")

	// ########## ENCRYPTED AGGREGATION ##########
	/*
		Agreg, encAgreg := getEncryptedAggregation(finalPrnus, numParties, weights)
		fmt.Print(Agreg[0][0])
		fmt.Print(encAgreg[0][0])

		fmt.Println("ECONDED AGGREGATION DONE")
	*/

	// ########## GET TEST IMAGES ########## (same camera (1) and different cameras (3))

	var nameCameras []string
	var dataTest = make(map[string][]gocv.Mat) // name of the camera : array of images

	dataTest, nameCameras = getDataTest(numExp, numCam, LENGTHIMGX, LENGTHIMGY) // read the images

	// ########## GET RESIDUALS ########## (of all test images)

	var residualsTest = make(map[string][][][][]PixelGray) // [layer B/G/R][image][rows][columns]

	for i := 0; i < numCam; i++ {
		residualsTest[nameCameras[i]] = getResiduals(dataTest[nameCameras[i]], nameCameras[i], LENGTHIMGX, LENGTHIMGY)
	}
	fmt.Println("RESIDUALS DONE")

	// ########## PREDICTION WITHOUT ENCODING ########## (scalar product between PRNU agregated and test image residual) => vector of SCORES

	pred := getPrediction(residualsTest, agreg, nameCameras)

	// write the scores in a text file
	writeLines(pred, "./experiment"+strconv.Itoa(numExp)+"/scores.txt")

	fmt.Println("PREDICTION DONE")

	// ########## ENCRYPTED PREDICTION ##########
	/*
			var numTestImages = len(residualsTest[nameCameras[0]][0])
			var finalResidual = make([][][]PixelGray, numTestImages) // [images][rows][columns]
			for i := 0; i < len(finalResidual); i++ {
				finalResidual[i] = make([][]PixelGray, LENGTHIMGX)
				for j := 0; j < len(finalResidual[i]); j++ {
					finalResidual[i][j] = make([]PixelGray, LENGTHIMGY)
				}
			}

			var numImg = 0
			for i := 0; i < len(nameCameras); i++ {
				for z := 0; z < len(residualsTest[nameCameras[i]][0]); z++ { // numImg
					finalResidual[numImg] = calculateFinalResidual(residualsTest[nameCameras[i]][0][z], residualsTest[nameCameras[i]][1][z], residualsTest[nameCameras[i]][2][z]) // combination of the 3 layers of each image
					numImg += 1
				}
			}
		                                                         // no se usa
			encPred, expRes := getEncryptedPrediction(finalPrnus[0], finalResidual, numTestImages) // we always use the 1st estimation of the 1st user
			writeLines(encPred, "./experiment"+strconv.Itoa(numExp)+"/encrypted.txt")
			writeLines(expRes, "./experiment"+strconv.Itoa(numExp)+"/claro.txt")
	*/

	fmt.Print("\n\n##############\n")
	fmt.Println("    FINISH")
	fmt.Print("##############\n\n")

}
