package main

func getPrediction(residualTest map[string][][][][]PixelGray, agreg [][]PixelGray, nameCameras []string) []float64 {

	//var scores = make([]PixelGray, len(residualTest[nameCameras[0]][0]))
	var scores []float64 // result vector from pediction (lenght = numImg)
	var finalResidual [][]PixelGray
	var result float64

	for i := 0; i < len(nameCameras); i++ { // numCam = 4
		//for j := 0; j < 3; j++ { // layer B,G,R
		for z := 0; z < len(residualTest[nameCameras[i]][0]); z++ { // numImg
			finalResidual = calculateFinalResidual(residualTest[nameCameras[i]][0][z], residualTest[nameCameras[i]][1][z], residualTest[nameCameras[i]][2][z]) // combination of the 3 layers of each image
			result = scalarProduct(finalResidual, agreg)
			//scores[z].pix = result
			scores = append(scores, result)
		}
	}

	return scores
}
