package main

import (
	hsv "code.google.com/p/sadbox/color"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/cmplx"
	"os"
	"runtime"
	"time"
	//"path"
)

const MAX = 2000

func renderImage(image *image.RGBA, a, b complex128) {
	bounds := image.Bounds()
	dx := bounds.Dx()
	dy := bounds.Dy()
	for x := 0; x < dx; x++ {
		for y := 0; y < dy; y++ {
			re := real(a) + (real(b)-real(a))*float64(x)/float64(dx)
			im := imag(a) + (imag(b)-imag(a))*float64(y)/float64(dy)
			c := complex(re, im)
			color := getColor(c)
			image.SetRGBA(x, y, color)
		}
	}
}

func getColor(c complex128) color.RGBA {
	z := complex(0, 0)
	for i := 0; i < MAX; i++ {
		z = z*z + c
		if cmplx.Abs(z) > 4 {
			r, g, b := hsv.HSVToRGB(float64((i*7)%360), 1, 1)
			return color.RGBA{r, g, b, 0xff}
		}
	}
	return color.RGBA{0, 0, 0, 0xff}
}

//concurrent stuff
type job struct {
	x, y int
	c    complex128
}
type result struct {
	x, y  int
	color color.RGBA
}

func jobFactory(a, b complex128, dx, dy int) chan job {
	jobs := make(chan job)
	go func() {
		for x := 0; x < dx; x++ {
			for y := 0; y < dy; y++ {
				re := real(a) + (real(b)-real(a))*float64(x)/float64(dx)
				im := imag(a) + (imag(b)-imag(a))*float64(y)/float64(dy)
				c := complex(re, im)
				jobs <- job{x, y, c}
			}
		}
		close(jobs)
	}()
	return jobs
}

func backgroundworker(jobs <-chan job, results chan<- result, done chan<- bool) {

	for job := range jobs {
		results <- result{job.x, job.y, getColor(job.c)}
	}
	done <- true
}

func resultCollector(image *image.RGBA, done chan<- bool) chan<- result {
	results := make(chan result)
	go func() {
		for result := range results {
			image.SetRGBA(result.x, result.y, result.color)
		}
		done <- true
	}()
	return results
}

func renderImageConcurrent(image *image.RGBA, a, b complex128) {
	bounds := image.Bounds()

	jobs := jobFactory(a, b, bounds.Dx(), bounds.Dy())
	done := make(chan bool)
	results := resultCollector(image, done)

	workerFactory(4, jobs, results)
	<-done
}

func workerFactory(count int, jobs <-chan job, results chan<- result) {
	done := make(chan bool)
	for i := 0; i < count; i++ {
		go backgroundworker(jobs, results, done)
	}
	go func() {
		for i := 0; i < count; i++ {
			<-done
		}
		close(results)
	}()
}

//main

func main() {

	numcpu := runtime.NumCPU()
	runtime.GOMAXPROCS(numcpu)

	f, _ := os.Create("logfile")
	log.SetOutput(f)
	log.Println("Number of used Cores: ", numcpu)

	im := image.NewRGBA(image.Rect(0, 0, 800, 600))
	concurrentImage := image.NewRGBA(image.Rect(0, 0, 800, 600))

	log.Println("rendering not concurrent stuff")
	t1 := time.Now()

	renderImage(im, -2.2-1.2i, 1+1.2i)

	log.Println(time.Since(t1), "")
	saveImage("image.png", im)

	log.Println("rendering concurrent")
	t2 := time.Now()

	renderImageConcurrent(concurrentImage, -2.2-1.2i, 1+1.2i)

	log.Println(time.Since(t2))

	saveImage("concurrent_image.png", concurrentImage)
	f.Close()
}

//#region helper functions

func saveImage(path string, i image.Image) {
	w, _ := os.Create(path)
	if err := png.Encode(w, i); err != nil {
		log.Println("Error writing image on disk")
		os.Exit(1)
	}
}

//#endregion
