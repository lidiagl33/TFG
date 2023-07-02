package main

import (
	"image"
)

func pixelArrayGray(img *image.Gray, size image.Point, lengthX int, lengthY int) [][]PixelGray {

	// convert the image into an array with pixel gray values [0,...,1]

	if (lengthX == 0) && (lengthY == 0) { // when the function is called by checkPRNU_2

		var pixels = make([][]PixelGray, size.Y)

		for i := 0; i < len(pixels); i++ {
			pixels[i] = make([]PixelGray, size.X)
		}

		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				pixels[y][x].pix = float64(img.GrayAt(x, y).Y)
			}
		}

		return pixels

	} else {

		// if the image is bigger than the established sized, it's cropped to that size
		if (size.X != lengthX) || (size.Y != lengthY) {

			var pixels = make([][]PixelGray, size.Y)

			for i := 0; i < len(pixels); i++ {
				pixels[i] = make([]PixelGray, size.X)
			}

			for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
				for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
					pixels[y][x].pix = float64(img.GrayAt(x, y).Y)
				}
			}

			var pixelsInRow = make([]float64, len(pixels)*len(pixels[0]))
			var changeRow1 = 0

			for i := 0; i < len(pixels); i++ {
				for j := 0; j < len(pixels[0]); j++ {
					pixelsInRow[i+j+changeRow1] = pixels[i][j].pix
				}
				changeRow1 += len(pixels[0]) - 1
			}

			var croppedPixels = make([][]PixelGray, lengthY)

			for i := 0; i < len(croppedPixels); i++ {
				croppedPixels[i] = make([]PixelGray, lengthX)
			}

			// when the dimensions selected (lengthX, lengthY) are less or equal than the original (3008x2000)

			for y := 0; y < lengthY; y++ {
				for x := 0; x < lengthX; x++ {
					croppedPixels[y][x].pix = pixels[y][x].pix
				}
			}

			// instead:

			/*var changeRow2 = 0
			for y := 0; y < lengthY; y++ {
				for x := 0; x < lengthX; x++ {
					croppedPixels[y][x].pix = pixelsInRow[y+x+changeRow2]
				}
				changeRow2 += lengthX - 1
			}*/

			return croppedPixels

			// if the image has the correct size
		} else {

			var pixels = make([][]PixelGray, size.Y)

			for i := 0; i < len(pixels); i++ {
				pixels[i] = make([]PixelGray, size.X)
			}

			for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
				for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
					pixels[y][x].pix = float64(img.GrayAt(x, y).Y)
				}
			}

			return pixels

		}
	}

}

func operateWithPixelsGray(pix1 [][]PixelGray, pix2 [][]PixelGray, check string) [][]PixelGray {

	// does the operation indicated by "check"

	var pixResult [][]PixelGray

	if len(pix1) >= len(pix2) {
		pixResult = make([][]PixelGray, len(pix1))
	} else {
		pixResult = make([][]PixelGray, len(pix2))
	}

	if len(pix1[0]) >= len(pix2[0]) {
		for i := 0; i < len(pix1); i++ {
			pixResult[i] = make([]PixelGray, len(pix1[0]))
		}
	} else {
		for i := 0; i < len(pix2); i++ {
			pixResult[i] = make([]PixelGray, len(pix2[0]))
		}
	}

	for i := 0; i < len(pixResult); i++ {
		for j := 0; j < len(pixResult[0]); j++ {

			if check == "+" {

				y := pix1[i][j].pix + pix2[i][j].pix

				pixResult[i][j].pix = y

			} else if check == "-" {

				y := pix1[i][j].pix - pix2[i][j].pix

				pixResult[i][j].pix = y

			} else if check == "*" {

				y := pix1[i][j].pix * pix2[i][j].pix

				pixResult[i][j].pix = y

			} else if check == "/" {

				var y float64

				if pix2[i][j].pix == 0 {
					y = pix1[i][j].pix / (pix2[i][j].pix + 0.000001) // make the denominator not zero
				} else {
					y = pix1[i][j].pix / pix2[i][j].pix
				}

				pixResult[i][j].pix = y
			}

		}
	}

	return pixResult
}
