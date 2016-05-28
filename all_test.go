package main

import(
	"fmt"
	"math"
	"testing"
)

type valuefqn func(x,y,z uint) float32
func xinc(x,y,z uint) float32 { return float32(x) }
func yinc(x,y,z uint) float32 { return float32(y) }
func zinc(x,y,z uint) float32 { return float32(z) }
func sphere(x,y,z uint) float32 {
	sum := float64(x*x + y*y + z*z)
	if math.Sqrt(sum) < 4.0 {
		return lerpf(0.0, 10.0, float32(math.Sqrt(sum)/4.0))
	}
	return 0.0
}

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

// evaluate the valfqn at each grid point; bail if it fails ever.
type valfqn func(x,y,z uint) bool
func validate(dims [3]uint, v valfqn) error {
	for z:=uint(0); z < dims[2]; z++ {
		for y:=uint(0); y < dims[1]; y++ {
			for x:=uint(0); x < dims[0]; x++ {
				if false == v(x,y,z) {
					return fmt.Errorf("failed at idx {%d,%d,%d}", x,y,z)
				}
			}
		}
	}
	return nil
}

// Identity: "resample" to the same space
func TestTrilinearIdentity(t *testing.T) {
	dims := [3]uint{4,4,4}
	in := make([]float32, dims[0]*dims[1]*dims[2])
	out := make([]float32, dims[0]*dims[1]*dims[2])

	if err := trilinearf(in,dims, out,dims) ; err != nil {
		t.Fatalf("trilinear failed: %v", err)
	}
	err := validate(dims, func(x,y,z uint) bool {
		linear := z*dims[1]*dims[0] + y*dims[0] + x
		return in[linear] == out[linear]
	})
	if err != nil {
		t.Fatalf("%v", err)
	}

	analytic(in, dims, xinc)
	if err := trilinearf(in,dims, out,dims) ; err != nil {
		t.Fatalf("trilinear failed: %v", err)
	}
	err = validate(dims, func(x,y,z uint) bool {
		linear := z*dims[1]*dims[0] + y*dims[0] + x
		return in[linear] == out[linear]
	})
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func TestTrilinear2x(t *testing.T) {
	idims := [3]uint{8,8,8}
	odims := [3]uint{16,16,16}
	in := make([]float32, idims[0]*idims[1]*idims[2])
	out := make([]float32, odims[0]*odims[1]*odims[2])

	if err := trilinearf(in,idims, out,odims); err != nil {
		t.Fatalf("null interp: %v", err)
	}

	analytic(in, idims, sphere)
	if err := trilinearf(in,idims, out,odims); err != nil {
		t.Fatalf("null interp: %v", err)
	}
	err := validate(odims, func(x,y,z uint) bool {
		sum := float64(x*x + y*y + z*z)
		if math.Sqrt(sum) < 4.0 {
			linear := z*odims[1]*odims[0] + y*odims[0] + x
			if out[linear] < 0.0 || out[linear] > 10.0 {
				return false
			}
		}
		return true
	})
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func TestTrilinear1pt5x(t *testing.T) {
	idims := [3]uint{8,8,8}
	odims := [3]uint{12,12,12}
	in := make([]float32, idims[0]*idims[1]*idims[2])
	out := make([]float32, odims[0]*odims[1]*odims[2])

	analytic(in, idims, sphere)
	if err := trilinearf(in,idims, out,odims); err != nil {
		t.Fatalf("null interp: %v", err)
	}
}

// Identity: "resample" to the same space
func TestTrilinearPlaneIdentity(t *testing.T) {
	dims := [3]uint{4,4,4}
	in := make([]float32, dims[0]*dims[1]*dims[2])
	out := make([]float32, dims[0]*dims[1]*dims[2])

	trilinear_planef(in,dims, out,dims)
	err := validate(dims, func(x,y,z uint) bool {
		linear := z*dims[1]*dims[0] + y*dims[0] + x
		return in[linear] == out[linear]
	})
	if err != nil {
		t.Fatalf("%v", err)
	}

	analytic(in, dims, xinc)
	if err := trilinearf(in,dims, out,dims) ; err != nil {
		t.Fatalf("trilinear failed: %v", err)
	}
	err = validate(dims, func(x,y,z uint) bool {
		linear := z*dims[1]*dims[0] + y*dims[0] + x
		return in[linear] == out[linear]
	})
	if err != nil {
		t.Fatalf("%v", err)
	}
}
