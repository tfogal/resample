package main

import(
	"errors"
	"fmt"
	"sync"
)

/*
[0,10) -> [0,100): 10/100 = .1/1
[0,496] -> [0,500): 496/500 = .992
*/
var id[3] uint
var od[3] uint

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

func trilinearf(in []float32,id [3]uint, out []float32, od [3]uint) error {
	if id[0] <= 1 || id[1] <= 1 || id[2] <= 1 {
		return errors.New("ill-defined results for small input volumes")
	}
	if od[0] <= 1 || od[1] <= 1 || id[2] <= 1 {
		return errors.New("ill-defined results for small output volumes")
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
	return nil
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
	// Now 2D iterate: our actual work is the entire plane for z=zoff
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

	wg.Add(int(od[2]))
	for z:=uint(0); z < od[2]; z++ {  // each iter starts a producer.
		if z == od[2]-1 {
			// hack: copy the last 2 planes twice just to satisfy arguments.  The
			// math works out such that "t" will always be 0.0, so only the first
			// plane will be read, but copy it to both planes just to be sure.
			blah := make([]float32, 2*id[1]*id[0])
			copy(blah, in[z*id[1]*id[0]:])
			copy(blah[id[1]*id[0]:], in[z*id[1]*id[0]:])
			go planef(blah, id, z, output, od, &wg)
		} else {
			go planef(in[z*od[1]*od[0]:], id, z, output, od, &wg)
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

func main() {
	fmt.Println("hi")
}
