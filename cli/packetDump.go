package cli

import "fmt"

func Dump(by []byte) {
	n := len(by)
	rowcount := 0
	stop := (n / 8) * 8
	k := 0
	for i := 0; i <= stop; i += 8 {
		k++
		if i+8 < n {
			rowcount = 8
		} else {
			rowcount = min(k*8, n) % 8
		}

		fmt.Printf("pos %02d  hex:  ", i)
		for j := 0; j < rowcount; j++ {
			fmt.Printf("%02x  ", by[i+j])
		}
		for j := rowcount; j < 8; j++ {
			fmt.Printf("    ")
		}
		fmt.Printf("  '%s'\n", viewString(by[i:(i+rowcount)]))
	}

}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func viewString(b []byte) string {
	r := []rune(string(b))
	for i := range r {
		if r[i] < 32 || r[i] > 126 {
			r[i] = '.'
		}
	}
	return string(r)
}
