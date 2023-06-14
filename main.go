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

	const LENGTHIMGX int = 2000 //3002 //3008
	const LENGTHIMGY int = 2000 //1994 //2000

	var numCam int = 1
	var numExp int = 1 // experiment
	var numParties int = 1
	//var imgPerCam int = 70
	//var imgExp int = 40
	//var imgTest int = 30

	var encryptedPediction = true

	if !encryptedPediction {
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

		// AGGREGATION WITHOUT ENCODING

		var agreg [][]PixelGray

		agreg = agregation(finalPrnus, numUsers, weights)
		fmt.Println("AGGREGATION DONE")

		fmt.Print(agreg[0][0])

	}

	// GET TEST IMAGES (same camera (1) and different cameras (3))

	var nameCameras []string
	var dataTest = make(map[string][]gocv.Mat) // name of the camera : array of images

	dataTest, nameCameras = getDataTest(numExp, numCam, LENGTHIMGX, LENGTHIMGY) // read the images

	// GET RESIDUALS (of all test images)

	var residualsTest = make(map[string][][][][]PixelGray) // [layer B/G/R][image][rows][columns]

	for i := 0; i < numCam; i++ {
		residualsTest[nameCameras[i]] = getResiduals(dataTest[nameCameras[i]], nameCameras[i], LENGTHIMGX, LENGTHIMGY)
	}
	fmt.Println("RESIDUALS DONE")

	// PREDICTION (scalar product between PRNU agregated and test image residual) => vector of SCORES

	///////ENCRYPTED PREDICTION////////////
	var numTestImages = 30                                   //len(residualsTest[nameCameras[0]][0]) + len(residualsTest[nameCameras[1]][0]) + len(residualsTest[nameCameras[2]][0]) + len(residualsTest[nameCameras[3]][0])
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
	var aux = make([][]PixelGray, 1)
	aux = append(aux, finalResidual[0]...)
	encPred, expRes := getEncryptedPrediction( /*finalPrnus[0]*/ aux, finalResidual, numTestImages) // we always use the 1st estimation of the 1st user
	writeLines(encPred, "./experiment"+strconv.Itoa(numExp)+"/encrypted2.txt")
	writeLines(expRes, "./experiment"+strconv.Itoa(numExp)+"/claro2.txt")
	/////////////////////////

	///////NORMAL PREDICTION////////////
	//pred := getPrediction(residualsTest, agreg, nameCameras)
	//writeLines(pred, "./experiment"+strconv.Itoa(numExp)+"/claro.txt")
	//fmt.Println("PREDICTION DONE")
	/////////////////////////

	// write the scores in a text file
	//writeLines(pred, "./experiment"+strconv.Itoa(numExp)+"/scores.txt")

	//fmt.Print(agreg[0][0], pred)

	fmt.Print("\n\n##############\n")
	fmt.Println("    FINISH")
	fmt.Print("##############\n\n")

}
