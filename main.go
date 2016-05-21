package main

import(
	"errors"
	"fmt"
)

/*
[0,10) -> [0,100): 10/100 = .1/1
[0,496] -> [0,500): 496/500 = .992
*/
var id[3] uint
var od[3] uint

func sample(in []float32,d [3]uint, l [8][3]uint, into *[8]float32) {
	if len(l) > len(into) {
		panic("buffer too small to hold all samples")
	}
	for i:=0; i < 8; i++ {
		vox := l[i]
		linear := vox[2]*d[0]*d[1] + vox[1]*d[0] + vox[0]
		if linear >= uint(len(in)) {
			fmt.Printf("bad: %d in %d-elem array", linear, len(in))
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
	s := [3]float32{float32(id[0])/float32(od[0]),
	                float32(id[1])/float32(od[1]),
	                float32(id[2])/float32(od[2])}
	var v [8]float32
	for z:=uint(0); z < od[2]; z++ {
		tz := s[2]*float32(z) - float32((uint(s[2])*z))
		for y:=uint(0); y < od[1]; y++ {
			ty := s[1]*float32(y) - float32((uint(s[1])*y))
			for x:=uint(0); x < od[0]; x++ {
				l := locations(s[0]*float32(x), s[1]*float32(y), s[2]*float32(z))
				for i:=0; i < 8; i++ {
					l[i] = max3u(l[i], [3]uint{id[0]-1, id[1]-1, id[2]-1})
				}
				sample(in, id, l, &v)
				// fractional part of x, i.e. how far between voxels in x
				tx := s[0]*float32(x) - float32((uint(s[0])*x))
				// front plane
				lowx := lerpf(v[0],v[1], tx)
				highx := lerpf(v[2],v[3], tx)
				bk_lowx := lerpf(v[4],v[5], tx)
				bk_highx := lerpf(v[6],v[7], tx)
				front := lerpf(lowx,highx, ty)
				back := lerpf(bk_lowx,bk_highx, ty)
				final := lerpf(front,back, tz)

				linear := z*od[1]*od[0] + y*od[0] + x
				out[linear] = final
			}
		}
	}
	return nil
}

func main() {
	fmt.Println("hi")
}
