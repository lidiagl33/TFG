package main

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

func calculatePRNU_1(layers []gocv.Mat, residuals *[][][]PixelGray, lengthX int, lengthY int) ([][][]PixelGray, [][][]PixelGray, []image.Point) {

	var originalImg image.Image
	var denoisedImg image.Image

	imgDenoised := gocv.NewMat()
	defer imgDenoised.Close()

	var arrayNum [][][]PixelGray // numerator [image][rows][columns]
	var arrayR [][][]PixelGray   // denominator

	var originalSizes []image.Point // store the sizes of the images

	var err error

	for i := 0; i < len(layers); i++ {

		// Y
		originalImg, err = layers[i].ToImage() // it will be coverted to image.Gray to have values between 0 and 1
		if err != nil {
			return nil, nil, nil
		}

		imgGray := image.NewGray(originalImg.Bounds())

		for y := originalImg.Bounds().Min.Y; y < originalImg.Bounds().Max.Y; y++ {
			for x := originalImg.Bounds().Min.X; x < originalImg.Bounds().Max.X; x++ {
				imgGray.Set(x, y, originalImg.At(x, y)) // Set already converts into color.Gray
			}
		}

		sizeOriginal := imgGray.Bounds().Size()
		originalSizes = append(originalSizes, sizeOriginal)

		// DENOISING

		gocv.FastNlMeansDenoising(layers[i], &imgDenoised)

		// X
		denoisedImg, err = imgDenoised.ToImage() // it will be coverted to image.Gray to have values between 0 and 1
		if err != nil {
			return nil, nil, nil
		}

		denoisedImgGray := image.NewGray(denoisedImg.Bounds())

		for y := denoisedImg.Bounds().Min.Y; y < denoisedImg.Bounds().Max.Y; y++ {
			for x := denoisedImg.Bounds().Min.X; x < denoisedImg.Bounds().Max.X; x++ {
				denoisedImgGray.Set(x, y, denoisedImg.At(x, y)) // Set already converts into color.Gray
			}
		}

		sizeDenoised := denoisedImgGray.Bounds().Size()

		// convert images into array of pixels (cropping the images)
		var pixOri [][]PixelGray = pixelArrayGray(imgGray, sizeOriginal, lengthX, lengthY)         // Y
		var pixDen [][]PixelGray = pixelArrayGray(denoisedImgGray, sizeDenoised, lengthX, lengthY) // X

		pixRes := operateWithPixelsGray(pixOri, pixDen, "-") // W=Y-X
		*residuals = append(*residuals, pixRes)

		pixNumerador := operateWithPixelsGray(pixRes, pixDen, "*") // W*X
		pixDivisor := operateWithPixelsGray(pixDen, pixDen, "*")   // R=X*X

		arrayNum = append(arrayNum, pixNumerador)
		arrayR = append(arrayR, pixDivisor)

	}

	return arrayNum, arrayR, originalSizes

}

func calculatePRNU_2(arrayNum [][][]PixelGray, arrayR [][][]PixelGray, maxLengthX int, maxLengthY int) ([][]PixelGray, [][]PixelGray) {

	// make the summations in numerator and denominator to get the result

	var pixSumNum = make([][]PixelGray, maxLengthY)
	for i := 0; i < len(pixSumNum); i++ {
		pixSumNum[i] = make([]PixelGray, maxLengthX)
	}
	var pixSumDen = make([][]PixelGray, maxLengthY)
	for i := 0; i < len(pixSumDen); i++ {
		pixSumDen[i] = make([]PixelGray, maxLengthX)
	}

	// len(arrayNum) == len(arrayR)
	for i := 0; i < len(arrayNum); i++ {
		for j := 0; j < len(arrayNum[i]); j++ {
			for k := 0; k < len(arrayNum[i][j]); k++ {
				pixSumNum[j][k].pix += arrayNum[i][j][k].pix
			}
		}
	}

	for i := 0; i < len(arrayR); i++ {
		for j := 0; j < len(arrayR[i]); j++ {
			for k := 0; k < len(arrayR[i][j]); k++ {
				pixSumDen[j][k].pix += arrayR[i][j][k].pix
			}
		}
	}

	return pixSumNum, pixSumDen
}

func calculateK(pixNum [][]PixelGray, pixDen [][]PixelGray) [][]PixelGray {

	// make the final division to get the estimated prnu

	var K [][]PixelGray

	K = operateWithPixelsGray(pixNum, pixDen, "/")

	return K

}

func calculateFinalK(K_B, K_G, K_R [][]PixelGray) [][]PixelGray {

	//0.3*K_R+0.59*K_G+0.11*K_B

	var finalK = make([][]PixelGray, len(K_B))
	for i := 0; i < len(finalK); i++ {
		finalK[i] = make([]PixelGray, len(K_B[0]))
	}

	for i := 0; i < len(finalK); i++ {
		for j := 0; j < len(finalK[0]); j++ {
			finalK[i][j].pix = 0.3*K_R[i][j].pix + 0.59*K_G[i][j].pix + 0.11*K_B[i][j].pix
		}
	}

	return finalK
}

func calculateResidual(layers []gocv.Mat, residuals *[][][]PixelGray, lengthX int, lengthY int) []image.Point {

	var originalImg image.Image
	var denoisedImg image.Image

	imgDenoised := gocv.NewMat()
	defer imgDenoised.Close()

	var originalSizes []image.Point // store the sizes of the images

	var err error

	for i := 0; i < len(layers); i++ {

		// Y
		originalImg, err = layers[i].ToImage() // it will be coverted to image.Gray to have values between 0 and 1
		if err != nil {
			return nil
		}

		imgGray := image.NewGray(originalImg.Bounds())

		for y := originalImg.Bounds().Min.Y; y < originalImg.Bounds().Max.Y; y++ {
			for x := originalImg.Bounds().Min.X; x < originalImg.Bounds().Max.X; x++ {
				imgGray.Set(x, y, originalImg.At(x, y)) // Set already converts into color.Gray
			}
		}

		sizeOriginal := imgGray.Bounds().Size()
		originalSizes = append(originalSizes, sizeOriginal)

		// DENOISING

		gocv.FastNlMeansDenoising(layers[i], &imgDenoised)

		// X
		denoisedImg, err = imgDenoised.ToImage() // it will be coverted to image.Gray to have values between 0 and 1
		if err != nil {
			return nil
		}

		denoisedImgGray := image.NewGray(denoisedImg.Bounds())

		for y := denoisedImg.Bounds().Min.Y; y < denoisedImg.Bounds().Max.Y; y++ {
			for x := denoisedImg.Bounds().Min.X; x < denoisedImg.Bounds().Max.X; x++ {
				denoisedImgGray.Set(x, y, denoisedImg.At(x, y)) // Set already converts into color.Gray
			}
		}

		sizeDenoised := denoisedImgGray.Bounds().Size()

		// convert images into array of pixels
		var pixOri [][]PixelGray = pixelArrayGray(imgGray, sizeOriginal, lengthX, lengthY)         // Y
		var pixDen [][]PixelGray = pixelArrayGray(denoisedImgGray, sizeDenoised, lengthX, lengthY) // X

		pixRes := operateWithPixelsGray(pixOri, pixDen, "-") // W=Y-X
		*residuals = append(*residuals, pixRes)

	}

	return originalSizes
}

func calculateFinalResidual(res_B, res_G, res_R [][]PixelGray) [][]PixelGray {

	//0.3*res_R+0.59*res_G+0.11*res_B

	var finalResidual = make([][]PixelGray, len(res_B))
	for i := 0; i < len(finalResidual); i++ {
		finalResidual[i] = make([]PixelGray, len(res_B[0]))
	}

	for i := 0; i < len(finalResidual); i++ {
		for j := 0; j < len(finalResidual[0]); j++ {
			finalResidual[i][j].pix = 0.3*res_R[i][j].pix + 0.59*res_G[i][j].pix + 0.11*res_B[i][j].pix
		}
	}

	return finalResidual
}

func checkPRNU_1(pixK [][]PixelGray, s string) {

	var mayor int
	var menor int

	for i := 0; i < len(pixK); i++ {
		for j := 0; j < len(pixK[i]); j++ {

			if pixK[i][j].pix > 1 {
				mayor++
			}
			if pixK[i][j].pix < -1 {
				menor++
			}
		}
	}

	fmt.Println(s)
	fmt.Printf("\tnumbers > 1: %d\n", mayor)
	fmt.Printf("\tnumbers < -1: %d\n\n", menor)
}

func checkPRNU_2(img *image.Gray, pixK [][]PixelGray) {

	size := img.Bounds().Size()

	pix := pixelArrayGray(img, size, 0, 0)

	res := scalarProduct(pix, pixK)

	fmt.Printf("\tResult: %f\n\n", res)

}
