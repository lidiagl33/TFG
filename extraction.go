package main

import (
	"C"

	"image"

	"gocv.io/x/gocv"
)

func extraction(images []gocv.Mat, userName string, lengthX int, lengthY int, check bool) [][][]PixelGray {

	numImg := len(images)

	var residualsB [][][]PixelGray // [image][rows][columns]
	var residualsG [][][]PixelGray
	var residualsR [][][]PixelGray

	var layersB []gocv.Mat // [layer B of the image 0, 1, ...]
	var layersG []gocv.Mat
	var layersR []gocv.Mat

	// iterate the images array to extract the channel of each image
	for i := 0; i < numImg; i++ {

		imgB := gocv.NewMat()
		imgG := gocv.NewMat()
		imgR := gocv.NewMat()

		gocv.ExtractChannel(images[i], &imgB, 0) // channel B
		gocv.ExtractChannel(images[i], &imgG, 1) // channel G
		gocv.ExtractChannel(images[i], &imgR, 2) // channel R

		layersB = append(layersB, imgB)
		layersG = append(layersG, imgG)
		layersR = append(layersR, imgR)

	}

	// CALCULATE THE NUMERATOR AND DENOMINATOR WITHOUT THE SUMMATION

	arrayNum1, arrayR1, _ := calculatePRNU_1(layersB, &residualsB, lengthX, lengthY) // B
	arrayNum2, arrayR2, _ := calculatePRNU_1(layersG, &residualsG, lengthX, lengthY) // G
	arrayNum3, arrayR3, _ := calculatePRNU_1(layersR, &residualsR, lengthX, lengthY) // R

	// in case there are images of different sizes, we will choose the bigger size to work with
	//maxLengthX, maxLengthY := calculateMaxLength(originalSizes)

	// SUMMATION OF THE NUMERATOR AND DENOMINATOR

	var pixSumNum1 = make([][]PixelGray, lengthY)
	for i := 0; i < len(pixSumNum1); i++ {
		pixSumNum1[i] = make([]PixelGray, lengthX)
	}
	var pixSumDen1 = make([][]PixelGray, lengthY)
	for i := 0; i < len(pixSumDen1); i++ {
		pixSumDen1[i] = make([]PixelGray, lengthX)
	}

	var pixSumNum2 = make([][]PixelGray, lengthY)
	for i := 0; i < len(pixSumNum2); i++ {
		pixSumNum2[i] = make([]PixelGray, lengthX)
	}
	var pixSumDen2 = make([][]PixelGray, lengthY)
	for i := 0; i < len(pixSumDen2); i++ {
		pixSumDen2[i] = make([]PixelGray, lengthX)
	}

	var pixSumNum3 = make([][]PixelGray, lengthY)
	for i := 0; i < len(pixSumNum3); i++ {
		pixSumNum3[i] = make([]PixelGray, lengthX)
	}
	var pixSumDen3 = make([][]PixelGray, lengthY)
	for i := 0; i < len(pixSumDen3); i++ {
		pixSumDen3[i] = make([]PixelGray, lengthX)
	}

	pixSumNum1, pixSumDen1 = calculatePRNU_2(arrayNum1, arrayR1, lengthX, lengthY)
	pixSumNum2, pixSumDen2 = calculatePRNU_2(arrayNum2, arrayR2, lengthX, lengthY)
	pixSumNum3, pixSumDen3 = calculatePRNU_2(arrayNum3, arrayR3, lengthX, lengthY)

	// CALCULATE K (ESTIMATION OF THE PRNU)

	pixK1 := calculateK(pixSumNum1, pixSumDen1) // PRNU B
	pixK2 := calculateK(pixSumNum2, pixSumDen2) // PRNU G
	pixK3 := calculateK(pixSumNum3, pixSumDen3) // PRNU R

	arrayPRNUs := [][][]PixelGray{pixK1, pixK2, pixK3}

	// VERIFY THE RESULTS (in case the flag "check" = true)

	if check {

		printResults(5, 0, userName)

		// CHECKING K (BETWEEN -1 Y 1)

		checkPRNU_1(pixK1, "PRNU B")
		checkPRNU_1(pixK2, "PRNU G")
		checkPRNU_1(pixK3, "PRNU R")

		// CHECKING K (SCALAR PRODUCT: IMAGE AND PRNU)

		printResults(0, 0, "")

		for i := 0; i < len(layersB); i++ {
			printResults(1, i, "")
			checkResults1(layersB[i], pixK1)
		}

		for i := 0; i < len(layersG); i++ {
			printResults(2, i, "")
			checkResults1(layersG[i], pixK2)
		}

		for i := 0; i < len(layersR); i++ {
			printResults(3, i, "")
			checkResults1(layersR[i], pixK3)
		}

		// CHECKING K (SCALAR PRODUCT: RESIDUAL AND PRNU)

		printResults(4, 0, "")

		for i := 0; i < len(residualsB); i++ {
			printResults(1, i, "")
			checkResults2(residualsB[i], pixK1)
		}

		for i := 0; i < len(residualsG); i++ {
			printResults(2, i, "")
			checkResults2(residualsG[i], pixK2)
		}

		for i := 0; i < len(residualsR); i++ {
			printResults(3, i, "")
			checkResults2(residualsR[i], pixK3)
		}

		// result with different image
		imgDif := gocv.IMRead("Images_MATLAB/Pxxx.jpg", gocv.IMReadColor)
		imgDifB := gocv.NewMat()
		imgDifG := gocv.NewMat()
		imgDifR := gocv.NewMat()
		gocv.ExtractChannel(imgDif, &imgDifB, 0)
		gocv.ExtractChannel(imgDif, &imgDifG, 1)
		gocv.ExtractChannel(imgDif, &imgDifR, 2)
		imgDifBDenoised := gocv.NewMat()
		imgDifGDenoised := gocv.NewMat()
		imgDifRDenoised := gocv.NewMat()
		gocv.FastNlMeansDenoising(imgDifB, &imgDifBDenoised)
		gocv.FastNlMeansDenoising(imgDifG, &imgDifGDenoised)
		gocv.FastNlMeansDenoising(imgDifR, &imgDifRDenoised)
		imB, err := imgDifB.ToImage()
		if err != nil {
			return nil
		}
		imG, err := imgDifB.ToImage()
		if err != nil {
			return nil
		}
		imR, err := imgDifB.ToImage()
		if err != nil {
			return nil
		}
		imgBGray := image.NewGray(imB.Bounds())
		imgGGray := image.NewGray(imG.Bounds())
		imgRGray := image.NewGray(imR.Bounds())
		for y := imB.Bounds().Min.Y; y < imB.Bounds().Max.Y; y++ {
			for x := imB.Bounds().Min.X; x < imB.Bounds().Max.X; x++ {
				imgBGray.Set(x, y, imB.At(x, y)) // Set already converts into color.Gray
			}
		}
		for y := imG.Bounds().Min.Y; y < imG.Bounds().Max.Y; y++ {
			for x := imG.Bounds().Min.X; x < imG.Bounds().Max.X; x++ {
				imgGGray.Set(x, y, imG.At(x, y)) // Set already converts into color.Gray
			}
		}
		for y := imR.Bounds().Min.Y; y < imR.Bounds().Max.Y; y++ {
			for x := imR.Bounds().Min.X; x < imR.Bounds().Max.X; x++ {
				imgRGray.Set(x, y, imR.At(x, y)) // Set already converts into color.Gray
			}
		}
		denoisedImgB, err := imgDifBDenoised.ToImage()
		if err != nil {
			return nil
		}
		denoisedImgG, err := imgDifGDenoised.ToImage()
		if err != nil {
			return nil
		}
		denoisedImgR, err := imgDifRDenoised.ToImage()
		if err != nil {
			return nil
		}
		denoisedBimgGray := image.NewGray(denoisedImgB.Bounds())
		denoisedGimgGray := image.NewGray(denoisedImgG.Bounds())
		denoisedRimgGray := image.NewGray(denoisedImgR.Bounds())
		for y := denoisedImgB.Bounds().Min.Y; y < denoisedImgB.Bounds().Max.Y; y++ {
			for x := denoisedImgB.Bounds().Min.X; x < denoisedImgB.Bounds().Max.X; x++ {
				denoisedBimgGray.Set(x, y, denoisedImgB.At(x, y)) // Set already converts into color.Gray
			}
		}
		for y := denoisedImgG.Bounds().Min.Y; y < denoisedImgG.Bounds().Max.Y; y++ {
			for x := denoisedImgG.Bounds().Min.X; x < denoisedImgG.Bounds().Max.X; x++ {
				denoisedGimgGray.Set(x, y, denoisedImgG.At(x, y)) // Set already converts into color.Gray
			}
		}
		for y := denoisedImgR.Bounds().Min.Y; y < denoisedImgR.Bounds().Max.Y; y++ {
			for x := denoisedImgR.Bounds().Min.X; x < denoisedImgR.Bounds().Max.X; x++ {
				denoisedRimgGray.Set(x, y, denoisedImgR.At(x, y)) // Set already converts into color.Gray
			}
		}

	}

	return arrayPRNUs
}

func getResiduals(images []gocv.Mat, camName string, lengthX int, lengthY int) [][][][]PixelGray {

	numImg := len(images)

	var residualsB [][][]PixelGray // [images][rows][columns]
	var residualsG [][][]PixelGray
	var residualsR [][][]PixelGray

	var layersB []gocv.Mat // [layer B of the image 0, 1, ...]
	var layersG []gocv.Mat
	var layersR []gocv.Mat

	// iterate the images array to extract the channel of each image
	for i := 0; i < numImg; i++ {

		imgB := gocv.NewMat()
		imgG := gocv.NewMat()
		imgR := gocv.NewMat()

		gocv.ExtractChannel(images[i], &imgB, 0) // channel B
		gocv.ExtractChannel(images[i], &imgG, 1) // channel G
		gocv.ExtractChannel(images[i], &imgR, 2) // channel R

		layersB = append(layersB, imgB)
		layersG = append(layersG, imgG)
		layersR = append(layersR, imgR)

	}

	// CALCULATE THE RESIDUAL

	calculateResidual(layersB, &residualsB, lengthX, lengthY) // B
	calculateResidual(layersG, &residualsG, lengthX, lengthY) // G
	calculateResidual(layersR, &residualsR, lengthX, lengthY) // R

	arrayResiduals := [][][][]PixelGray{residualsB, residualsG, residualsR} // [B,G,R][image][rows][columns]

	return arrayResiduals

}
