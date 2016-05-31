package main

import(
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"reflect"
	"sync"
	"unsafe"
)

var idims [3]uint // input dimensions
var odims [3]uint // output dimensions
var mksphere = false
var input string
var output string

func init() {
	flag.BoolVar(&mksphere, "sphere", false, "create 'sphere' output file")
	flag.UintVar(&idims[0], "ix", 0, "dimensions of the input file, X")
	flag.UintVar(&idims[1], "iy", 0, "dimensions of the input file, Y")
	flag.UintVar(&idims[2], "iz", 0, "dimensions of the input file, Z")
	flag.UintVar(&odims[0], "ox", 0, "dimensions of the output file, X")
	flag.UintVar(&odims[1], "oy", 0, "dimensions of the output file, Y")
	flag.UintVar(&odims[2], "oz", 0, "dimensions of the output file, Z")
	flag.StringVar(&input, "input", "", "file to read from")
	flag.StringVar(&output, "output", "", "file to create")
}

func validate_args() error {
	if odims[0]*odims[1]*odims[2] == 0 {
		return errors.New("output volume size is 0.")
	}
	if output == "" {
		return errors.New("output filename not given")
	}
	if !mksphere && idims[0]*idims[1]*idims[2] == 0 {
		return errors.New("input volume size is 0.")
	}
	if !mksphere && input == "" {
		return errors.New("input filename not given")
	}
	return nil
}

func sample(in []float32,d [3]uint, l [8][3]uint, into *[8]float32) {
	for i:=0; i < 8; i++ {
		vox := l[i]
		linear := vox[2]*d[1]*d[0] + vox[1]*d[0] + vox[0]
		if linear >= uint(len(in)) {
			fmt.Printf("bad: %d in %d-elem array\n", linear, len(in))
			fmt.Printf("b: {%d,%d,%d} in {%d,%d,%d}\n", vox[0],vox[1],vox[2],
			           d[0],d[1],d[2]);
			panic("linear too large")
		}
		into[i] = in[linear]
	}
}

func locations(x, y, z float32) [8][3]uint {
	return [8][3]uint{
		[3]uint{uint(x)+0, uint(y)+0, uint(z)+0}, // 0
		[3]uint{uint(x)+1, uint(y)+0, uint(z)+0}, // 1
		[3]uint{uint(x)+0, uint(y)+1, uint(z)+0}, // 2
		[3]uint{uint(x)+1, uint(y)+1, uint(z)+0}, // 3
		[3]uint{uint(x)+0, uint(y)+0, uint(z)+1}, // 4
		[3]uint{uint(x)+1, uint(y)+0, uint(z)+1}, // 5
		[3]uint{uint(x)+0, uint(y)+1, uint(z)+1}, // 6
		[3]uint{uint(x)+1, uint(y)+1, uint(z)+1}, // 7
	}
}

func lerpf(a,b float32, t float32) float32 {
	return (1-t)*a + t*b
}

func max(a uint, mx uint) uint {
	if a > mx {
		return mx
	}
	return a
}
func max3u(l [3]uint, mx [3]uint) [3]uint {
	return [3]uint{max(l[0], mx[0]), max(l[1], mx[1]), max(l[2], mx[2])}
}

func trilinearf(in []float32,id [3]uint, out []float32, od [3]uint) {
	if id[0] <= 1 || id[1] <= 1 || id[2] <= 1 {
		panic("ill-defined results for small input volumes")
	}
	if od[0] <= 1 || od[1] <= 1 || id[2] <= 1 {
		panic("ill-defined results for small output volumes")
	}
	if uint(len(in)) < id[0]*id[1]*id[2] {
		panic("input too small")
	}
	if uint(len(out)) < od[0]*od[1]*od[2] {
		panic("output too small")
	}
	ratio := [3]float32{float32(id[0]) / float32(od[0]),
	                    float32(id[1]) / float32(od[1]),
	                    float32(id[2]) / float32(od[2])}
	var v [8]float32
	var lower [3]uint // lower index, bound
	var upper [3]uint // higher index, or bound
	var mid [3]float32 // where we are through the input volume
	var t [3]float32 // the fractional part we are between 'lower' and 'upper'
	for z:=uint(0); z < od[2]; z++ {
		mid[2] = float32(z) * ratio[2]
		lower[2] = uint(mid[2])
		// max: ensure we don't exceed the bounds of the input data
		upper[2] = max(lower[2]+1, id[2]-1)
		t[2] = mid[2] - float32(lower[2])
		for y:=uint(0); y < od[1]; y++ {
			mid[1] = float32(y) * ratio[1]
			lower[1] = uint(mid[1])
			upper[1] = max(lower[1]+1, id[1]-1)
			t[1] = mid[1] - float32(lower[1])

			for x:=uint(0); x < od[0]; x++ {
				mid[0] = float32(x) * ratio[0]
				lower[0] = uint(mid[0])
				upper[0] = max(lower[0]+1, id[0]-1)
				t[0] = mid[0] - float32(lower[0])
				l := [8][3]uint{ // locations
					[3]uint{lower[0], lower[1], lower[2]},
					[3]uint{upper[0], lower[1], lower[2]},
					[3]uint{lower[0], upper[1], lower[2]},
					[3]uint{upper[0], upper[1], lower[2]},
					[3]uint{lower[0], lower[1], upper[2]},
					[3]uint{upper[0], lower[1], upper[2]},
					[3]uint{lower[0], upper[1], upper[2]},
					[3]uint{upper[0], upper[1], upper[2]},
				}

				sample(in, id, l, &v)
				// front plane
				lowx := lerpf(v[0],v[1], t[0])
				highx := lerpf(v[2],v[3], t[0])
				bk_lowx := lerpf(v[4],v[5], t[0])
				bk_highx := lerpf(v[6],v[7], t[0])
				front := lerpf(lowx,highx, t[1])
				back := lerpf(bk_lowx,bk_highx, t[1])
				final := lerpf(front,back, t[2])

				linear := z*od[1]*od[0] + y*od[0] + x
				out[linear] = final
			}
		}
	}
}

type scanline struct {
	s uint
	data []float32
}

// trilinear interpolation using 2 planes of data.
// The input is 2 planes of what is expected to be a larger dataset.
// ZOFF describes where the lower of those 2 planes begins within the large
// dataset.
func planef(in []float32,id [3]uint, zoff uint, out chan *scanline, od [3]uint,
            wg *sync.WaitGroup) {
	defer wg.Done()
	if id[0] <= 1 || id[1] <= 1 || id[2] <= 1 {
		panic("ill-defined results for small input volumes")
	}
	if od[0] <= 1 || od[1] <= 1 || id[2] <= 1 {
		panic("ill-defined results for small output volumes")
	}
	if uint(len(in)) < id[1]*id[0]*2 {
		panic("input length too small")
	}
	ratio := [3]float32{float32(id[0]) / float32(od[0]),
	                    float32(id[1]) / float32(od[1]),
	                    float32(id[2]) / float32(od[2])}
	var v [8]float32
	var lower [3]uint // lower index, bound
	var upper [3]uint // higher index, or bound
	var mid [3]float32 // where we are through the input volume
	var t [3]float32 // the fractional part we are between 'lower' and 'upper'

	// These don't matter much for the z dimension, because we only process a
	// single plane at this point.  But just to keep it orthogonal to the other
	// dimensions, let's calculate it anyway ...
	mid[2] = float32(zoff) * ratio[2]
	lower[2] = uint(mid[2])
	// max: ensure we don't exceed the bounds of the input data
	upper[2] = max(lower[2]+1, id[2]-1)
	t[2] = mid[2] - float32(lower[2])

	// We send a scanline-at-a-time to the channel.  Though a single value would
	// work fine, no consumer would want so little data at a time.
	sline := make([]float32, od[0])
	// Now 2D iterate: our work is the entire plane for z=zoff
	for y:=uint(0); y < od[1]; y++ {
		mid[1] = float32(y) * ratio[1]
		lower[1] = uint(mid[1])
		upper[1] = max(lower[1]+1, id[1]-1)
		t[1] = mid[1] - float32(lower[1])

		for x:=uint(0); x < od[0]; x++ {
			mid[0] = float32(x) * ratio[0]
			lower[0] = uint(mid[0])
			upper[0] = max(lower[0]+1, id[0]-1)
			t[0] = mid[0] - float32(lower[0])
			l := [8][3]uint{ // locations
				[3]uint{lower[0], lower[1], 0}, // z is always 0, or 1
				[3]uint{upper[0], lower[1], 0}, // because we only expect to have
				[3]uint{lower[0], upper[1], 0}, // two planes loaded.
				[3]uint{upper[0], upper[1], 0}, // those planes are likely to be
				[3]uint{lower[0], lower[1], 1}, // somewhere in the middle of the
				[3]uint{upper[0], lower[1], 1}, // dataset, though.
				[3]uint{lower[0], upper[1], 1},
				[3]uint{upper[0], upper[1], 1},
			}

			sample(in, id, l, &v)
			lowx := lerpf(v[0],v[1], t[0]) // front plane
			highx := lerpf(v[2],v[3], t[0])
			bk_lowx := lerpf(v[4],v[5], t[0]) // back plane
			bk_highx := lerpf(v[6],v[7], t[0])
			front := lerpf(lowx,highx, t[1])
			back := lerpf(bk_lowx,bk_highx, t[1])
			final := lerpf(front,back, t[2])

			sline[x] = final
		}
		s := scanline { s:zoff*od[1]+y, data: make([]float32, od[0]) }
		copy(s.data, sline)
		out <- &s
	}
}

func trilinear_planef(in []float32, id [3]uint, out []float32, od [3]uint) {
	output := make(chan *scanline)
	var wg sync.WaitGroup

	ratio := float32(id[2]) / float32(od[2])
	wg.Add(int(od[2]))
	for z:=uint(0); z < od[2]; z++ {  // each iter starts a producer.
		mid := float32(z) * ratio
		lower := uint(mid)
		if lower == id[2]-1 {
			// hack: copy the last 2 planes twice.  This situation occurs when we
			// have so many more planes in the output dataset that we are creating
			// "N" planes for every plane in the input dataset.  If that's the case
			// then the *last* plane will be duplicated N times, too, but there isn't
			// a plane on the "upper" (lower+1) side to source the data from.
			// We duplicate the input's final plane as the plane on both sides.  When
			// the interpolation happens, the 't' will be irrelevant since both sides
			// are the same.
			blah := make([]float32, 2*id[1]*id[0])
			copy(blah, in[lower*id[1]*id[0]:])
			copy(blah[id[1]*id[0]:], in[lower*id[1]*id[0]:])
			go planef(blah, id, z, output, od, &wg)
		} else {
/*
			fmt.Printf("len(in)=%d, z=%d, lower=%d, id={%d,%d,%d}\n", len(in), z,
			           lower, id[0],id[1],id[2])
*/
			go planef(in[lower*id[1]*id[0]:], id, z, output, od, &wg)
		}
	}
	done := make(chan int)
	go func() { // consumer.
		for sline := range output {
			if uint(len(sline.data)) != od[0] {
				fmt.Printf("len(scanline)=%d, != %d\n", len(sline.data), od[0])
				panic("scanline length broken")
			}
			s := sline.s
			copy(out[s*od[0]:(s+1)*od[0]], sline.data)
		}
		done <- 1
	}()
	wg.Wait()
	close(output)
	<-done
	close(done)
}

type valuefqn func(x,y,z uint) float32
// define the volume according to the given function
func analytic(data []float32, dims [3]uint, value valuefqn) {
	for z:=uint(0); z < dims[2]; z++ {
		for y:=uint(0); y < dims[1]; y++ {
			for x:=uint(0); x < dims[0]; x++ {
				linear := z*dims[1]*dims[0] + y*dims[0] + x
				data[linear] = value(x,y,z)
			}
		}
	}
}
func sqrtf(a float32) float32 { // go doesn't provide a sqrt for float32.
	return float32(math.Sqrt(float64(a)))
}
func dist(a,b,c float32, x,y,z float32) float32 {
	return sqrtf((a-x)*(a-x) + (b-y)*(b-y) + (c-z)*(c-z))
}
func sphere(x,y,z uint) float32 {
	d := dist(float32(x),float32(y),float32(z), 32,32,32)
	if d < 8.0 {
		return lerpf(0.0, 10.0, d / 8.0)
		//return lerpf(0.0, 10.0, float32(math.Sqrt(sum)/4.0))
	}
	return 0.0
}

func castfb(data []float32) []byte {
	var f32 float32 // needed because unsafe.Sizeof takes an expr, not a type.
	hdr := *(*reflect.SliceHeader)(unsafe.Pointer(&data))
	hdr.Len = int(unsafe.Sizeof(f32))*len(data)
	hdr.Cap = int(unsafe.Sizeof(f32))*len(data)
	asbytes := *(*[]byte)(unsafe.Pointer(&hdr))
	return asbytes
}
func castbf(data []byte) []float32 {
	var f32 float32 // needed because unsafe.Sizeof takes an expr, not a type.
	hdr := *(*reflect.SliceHeader)(unsafe.Pointer(&data))
	hdr.Len = len(data) / int(unsafe.Sizeof(f32))
	hdr.Cap = len(data) / int(unsafe.Sizeof(f32))
	asfloats := *(*[]float32)(unsafe.Pointer(&hdr))
	return asfloats
}

func create_sphere(fname string, dims [3]uint) error {
	data := make([]float32, dims[0]*dims[1]*dims[2])
	analytic(data, dims, sphere)

	f, err := os.OpenFile(fname, os.O_CREATE | os.O_TRUNC | os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("cannot create %s: %v\n", fname, err)
	}

	asbytes := castfb(data)
	bytes, err := f.Write(asbytes)
	if err != nil || uint(bytes) != dims[0]*dims[1]*dims[2]*4 {
		f.Close()
		os.Remove(fname)
		return fmt.Errorf("error writing %s: %v\n", fname, err)
	}

	if err := f.Close() ; err != nil {
		os.Remove(fname)
		return fmt.Errorf("error closing %s: %v\n", fname, err)
	}
	return nil
}

func read_rawf(fname string, dims [3]uint) ([]float32, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var float float32
	sz := dims[0]*dims[1]*dims[2]*uint(unsafe.Sizeof(float))
	data := make([]byte, sz)
	i := 0
	for uint(i) < sz {
		bytes, err := f.Read(data[i:])
		if err != nil {
			return nil, err
		}
		i += bytes
	}

	return castbf(data), nil
}

func write_rawf(fname string, data []float32, dims [3]uint) error {
	f, err := os.OpenFile(fname, os.O_CREATE | os.O_TRUNC | os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	var float float32
	sz := dims[0]*dims[1]*dims[2]*uint(unsafe.Sizeof(float))
	asbytes := castfb(data)
	i := 0
	for uint(i) < sz {
		bytes, err := f.Write(asbytes)
		if err != nil {
			f.Close()
			os.Remove(fname)
			return fmt.Errorf("error writing %s: %v", fname, err)
		}
		i += bytes
	}

	if err := f.Close() ; err != nil {
		os.Remove(fname)
		return fmt.Errorf("error closing %s: %v", fname, err)
	}
	return nil
}

func main() {
	flag.Parse()
	if err := validate_args() ; err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	in, err := read_rawf(input, idims)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading data: %v\n", err)
		return
	}
	out := make([]float32, odims[0]*odims[1]*odims[2])

	trilinearf(in, idims, out, odims)
	if err := write_rawf(output, out, odims) ; err != nil {
		fmt.Fprintf(os.Stderr, "writing data: %v\n", err)
	}
}
