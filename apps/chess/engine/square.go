// engine/square.go
package engine

func squareToAlgebraic(sq uint8) string {
	file := sq % 8
	rank := sq / 8
	return string([]byte{
		'a' + file,
		'1' + rank,
	})
}
