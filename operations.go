package main

import (
	"fmt"
	"image"
	"math"

	"github.com/montanaflynn/stats"
	"gocv.io/x/gocv"
)

func scalarProduct(pix, k [][]PixelGray) float64 {

	var result float64

	// TO CHECK
	/*
		for i := 0; i < len(pix); i++ {
			for j := 0; j < len(pix[0]); j++ {
				pix[i][j].pix = 1
			}
		}

		for i := 0; i < len(k); i++ {
			for j := 0; j < len(k[0]); j++ {
				k[i][j].pix = 1
			}
		}
	*/

	pixMult := operateWithPixelsGray(pix, k, "*")

	for i := 0; i < len(pixMult); i++ {
		for j := 0; j < len(pixMult[0]); j++ {
			result += pixMult[i][j].pix
		}
	}

	return result
}

func calculateMaxLength(originalSizes []image.Point) (int, int) {

	// calculate the maximum lenght of the images among all of them

	var maxLenghtX int
	var maxLenghtY int

	for i := 0; i < len(originalSizes); i++ {
		if i == 0 {
			maxLenghtX = originalSizes[i].X
			maxLenghtY = originalSizes[i].Y
		} else {
			if originalSizes[i].X > maxLenghtX {
				maxLenghtX = originalSizes[i].X
			}
			if originalSizes[i].Y > maxLenghtY {
				maxLenghtY = originalSizes[i].Y
			}
		}
	}

	return maxLenghtX, maxLenghtY
}

func convertToGray(img image.Image) *image.Gray {

	// convert an RGBA image into Gray image

	imgGray := image.NewGray(img.Bounds())

	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			imgGray.Set(x, y, img.At(x, y))
		}
	}

	return imgGray

}

func checkResults1(img gocv.Mat, pixK [][]PixelGray) {

	imgIm, err := img.ToImage()
	if err != nil {
		return
	}

	imgImGray := convertToGray(imgIm)

	checkPRNU_2(imgImGray, pixK)

}

func checkResults2(residual [][]PixelGray, pixK [][]PixelGray) {

	res := scalarProduct(residual, pixK)

	fmt.Printf("\tResult: %f\n\n", res)

}

func checkResults3(res [][]float64, agreg [][]PixelGray, averageK float64, layer string) {

	// res => with encryption
	// agreg => without encryption

	absoluteError := make([][]float64, len(res)) // len(res) == len(agreg)
	for i := range res {
		absoluteError[i] = make([]float64, len(res[0]))
	}

	for i := 0; i < len(res); i++ {
		for j := 0; j < len(res[0]); j++ {
			absoluteError[i][j] = math.Abs(agreg[i][j].pix - res[i][j])
		}
	}

	var averageAbsoluteError, relativeError float64

	for i := 0; i < len(absoluteError); i++ {
		for j := 0; j < len(absoluteError[0]); j++ {
			averageAbsoluteError += absoluteError[i][j]
		}
	}

	averageAbsoluteError = averageAbsoluteError / float64((len(absoluteError) * len(absoluteError[0])))

	relativeError = averageAbsoluteError / averageK

	//var absoluteError float64
	/*var averageRelativeError float64

	// calculate de average error: add up all values / number of values
	/*for i := 0; i < len(errorEncrypted); i++ {
		for j := 0; j < len(errorEncrypted[0]); j++ {
			absoluteError += errorEncrypted[i][j]
		}
	}*/

	//averageAbsoluteError = averageAbsoluteError / float64((len(errorEncrypted) * len(errorEncrypted[0])))
	//averageRelativeError = absoluteError / averageAgreg

	/*relativeError := make([][]float64, len(absoluteError)) // len(res) == len(agreg)
	for i := range res {
		relativeError[i] = make([]float64, len(absoluteError[0]))
	}

	for i := 0; i < len(absoluteError); i++ {
		for j := 0; j < len(absoluteError[0]); j++ {
			relativeError[i][j] = absoluteError[i][j] / averageAgreg
		}
	}

	for i := 0; i < len(relativeError); i++ {
		for j := 0; j < len(relativeError[0]); j++ {
			averageRelativeError += relativeError[i][j]
		}
	}

	averageRelativeError = averageRelativeError / float64((len(relativeError) * len(relativeError[0])))*/

	fmt.Printf("ERROR from layer %s OBTAINED = %f\n", layer, relativeError)

}

func printResults(c, index int, userName string) {

	if c == 0 {
		fmt.Print("\n")
		fmt.Println("\t--- SCALAR PRODUCT WITH PRNUs ---")
		fmt.Print("\n")
	} else if c == 1 {
		fmt.Printf("\t--- checking prnu B%d ---", index+1)
		fmt.Print("\n")
	} else if c == 2 {
		fmt.Printf("\t--- checking prnu G%d ---", index+1)
		fmt.Print("\n")
	} else if c == 3 {
		fmt.Printf("\t--- checking prnu R%d ---", index+1)
		fmt.Print("\n")
	} else if c == 4 {
		fmt.Print("\n")
		fmt.Println("\t--- SCALAR PRODUCT WITH RESIDUALS ---")
		fmt.Print("\n")
	} else if c == 5 {
		fmt.Printf("\t-.-.-.-.-.-.-.-.- USER: %q -.-.-.-.-.-.-.-.-\n\n", userName)
	}

}

func calculateVariance(finalPrnus [][][]PixelGray, numParties int) []float64 {

	var variance []float64 // one variance per user, len(variance)=numParties

	// numParties = len(finalPrnus)
	for i := 0; i < numParties; i++ {
		var aux []float64
		for j := 0; j < len(finalPrnus[i]); j++ {
			for z := 0; z < len(finalPrnus[i][j]); z++ {
				aux = append(aux, finalPrnus[i][j][z].pix)
			}
		}
		auxVariance, err := stats.Variance(aux)
		if err != nil {
			return nil
		}
		variance = append(variance, auxVariance)
	}

	return variance
}
