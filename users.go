package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"gocv.io/x/gocv"
)

func getDataExp(numParties int, numExp int, lengthX int, lengthY int) (map[string][]gocv.Mat, int, []string) {

	var data = make(map[string][]gocv.Mat) // [userName][arrayOfImages]
	var numUsers int
	var nameUsers []string
	//var correctSize = image.Point{lengthX, lengthY}

	// 1, 2, 5, 10 users/parties
	for i := 0; i < numParties; i++ {

		var filesNames []string
		var images []gocv.Mat

		nameDir := "experiment" + strconv.Itoa(numExp) + "/user" + strconv.Itoa(i+1) // we assume the format "user1" for the folders containing the images
		nameUsers = append(nameUsers, nameDir)

		if fileExists(nameDir) {

			numUsers++

			dir, err := os.Open(nameDir)
			if err != nil {
				fmt.Println(err)
				return nil, 0, nil
			}

			files, err := dir.Readdir(0) // read all the files inside the folder
			if err != nil {
				fmt.Println(err)
				return nil, 0, nil
			}

			for j := 0; j < len(files); j++ {
				if files[j].Name() != "desktop.ini" {
					filesNames = append(filesNames, files[j].Name())
				}
			}

			for z := 0; z < len(filesNames); z++ {
				fmt.Printf("\tLoading %q\n", nameDir+"/"+filesNames[z])
				img := gocv.IMRead(nameDir+"/"+filesNames[z], gocv.IMReadColor) // read the image

				imgAux, err := img.ToImage()
				if err != nil {
					return nil, 0, nil
				}
				size := imgAux.Bounds().Size()

				// we are working with X > Y
				if size.Y > size.X {
					gocv.Rotate(img, &img, gocv.Rotate90Clockwise)
				}

				images = append(images, img)
			}

			data[nameDir] = images

			fmt.Printf("\nImages of %q loaded\n\n", nameDir)

		}

	}

	return data, numUsers, nameUsers

}

func getDataTest(numExp int, numCam int, lengthX int, lengthY int) (map[string][]gocv.Mat, []string) {

	var data = make(map[string][]gocv.Mat) // [userName][arrayOfImages]
	var nameCameras []string
	//var correctSize = image.Point{lengthX, lengthY}

	var labels []float64 // matlab function ROC curve

	// 4 cameras (1 TRUE, 3 FALSE)
	for i := 0; i < numCam; i++ {

		var filesNames []string
		var images []gocv.Mat

		nameDir := "experiment" + strconv.Itoa(numExp) + "/test" + "/camera" + strconv.Itoa(i+1) // we assume the format "camera1" for the folders containing the images
		nameCameras = append(nameCameras, nameDir)

		if fileExists(nameDir) {

			dir, err := os.Open(nameDir)
			if err != nil {
				fmt.Println(err)
				return nil, nil
			}

			files, err := dir.Readdir(0) // read all the files inside the folder
			if err != nil {
				fmt.Println(err)
				return nil, nil
			}

			for j := 0; j < len(files); j++ {
				if files[j].Name() != "desktop.ini" {
					filesNames = append(filesNames, files[j].Name())
				}
			}

			for z := 0; z < len(filesNames); z++ {
				fmt.Printf("\tLoading %q\n", nameDir+"/"+filesNames[z])
				img := gocv.IMRead(nameDir+"/"+filesNames[z], gocv.IMReadColor) // read the image

				imgAux, err := img.ToImage()
				if err != nil {
					return nil, nil
				}
				size := imgAux.Bounds().Size()

				// we are working with X > Y
				if size.Y > size.X {
					gocv.Rotate(img, &img, gocv.Rotate90Clockwise)
				}

				images = append(images, img)

				if numExp == i+1 { // camera TRUE
					labels = append(labels, 1)
				} else { // camera FALSE
					labels = append(labels, 0)
				}

			}

			data[nameDir] = images

			fmt.Printf("\nImages of %q loaded\n\n", nameDir)

		}

	}

	writeLines(labels, "./experiment"+strconv.Itoa(numExp)+"/labels.txt")

	return data, nameCameras

}

func fileExists(rute string) bool {

	_, err := os.Stat(rute)

	if os.IsNotExist(err) {
		return false
	}

	return true
}

// writes the lines to the given file
func writeLines(lines []float64, path string) error {

	file, err := os.Create(path)

	if err != nil {
		return err
	}

	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}

	return w.Flush()

}

func readLines(path string) []float64 {

	var lines []float64

	file, err := os.Open(path)

	if err != nil {
		fmt.Println("Error in path, reading weights file")
	}

	defer file.Close()

	s := bufio.NewScanner(file)

	for s.Scan() {
		data, err := strconv.ParseFloat(s.Text(), 64)
		if err != nil {
			fmt.Println(err)
		}
		lines = append(lines, data)
	}

	return lines

}
