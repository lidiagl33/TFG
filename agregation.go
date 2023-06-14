package main

func agregation(prnus [][][]PixelGray, numUsers int, weights []float64) [][]PixelGray {

	// OPERATION TO MAKE: addition PRNUs / number of users

	var numerator = make([][]PixelGray, len(prnus[0])) // summation prnus
	for i := 0; i < len(numerator); i++ {
		numerator[i] = make([]PixelGray, len(prnus[0][0]))
	}

	for i := 0; i < len(prnus); i++ { // len(prnus) = numUsers
		for j := 0; j < len(prnus[i]); j++ {
			for k := 0; k < len(prnus[i][j]); k++ {
				numerator[j][k].pix += weights[i] * prnus[i][j][k].pix
			}
		}
	}

	var result = make([][]PixelGray, len(numerator))
	for i := 0; i < len(result); i++ {
		result[i] = make([]PixelGray, len(numerator[0]))
	}

	for i := 0; i < len(result); i++ {
		for j := 0; j < len(result[0]); j++ {
			result[i][j].pix = numerator[i][j].pix / float64(numUsers) // final values
		}
	}

	return result

}
