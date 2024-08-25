package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"sync"
	"unsafe"
)

var (
	possible = make(map[shape]struct{}) //400_000_000

	possibleList      []shape
	possibleLeftList  []shape
	possibleRightList []shape

	additions  = make([][]shape, 8)
	stackables = []shape{}

	changed = true
)

func addPossible(s shape) {
	if _, ok := possible[s]; !ok {
		addPossibleNoCheck(s)
	}
}

func addPossibleNoCheck(s shape) {
	possible[s] = struct{}{}
	possibleList = append(possibleList, s)
	changed = true

	if s&0b1100_1100_1100_1100_1100_1100_1100_1100 == 0 {
		possibleLeftList = append(possibleLeftList, s)
	}
	if s&^0b1100_1100_1100_1100_1100_1100_1100_1100 == 0 {
		possibleRightList = append(possibleRightList, s)
	}
}

func main() {
	if _, err := os.Stat("./possible-sorted.bin"); os.IsNotExist(err) {
		findAllPossibleShapes()
	} else {
		readPossibleShapes()
	}

	fmt.Println("Possible shapes:", len(possibleList))

	// slices.Sort(possibleList)
	// fmt.Println("Sorted possible shapes:", len(possibleList))

	// pointer := unsafe.SliceData(possibleList)
	// bytes := unsafe.Slice((*byte)(unsafe.Pointer(pointer)), len(possibleList)*4)
	// file, _ := os.Create("possible-sorted.bin")
	// defer file.Close()
	// file.Write(bytes)

	// count := 0
	// count2 := 0

	// for i := range 0b1_0000_0000_0000_0000_0000_0000 {
	// 	s := shape(i&0b1111_1111 | ((i & 0b1111_1111_0000_0000) << 8))
	// 	if s.isPossible() {
	// 		count++
	// 	}
	// }

	// for _, s := range possibleList {
	// 	x := s
	// 	y := shape(0)
	// 	i := 0
	// 	for x.hasCrystal() {
	// 		b, t := x.unstackBottom()
	// 		x = t
	// 		y |= b << (i * 4)
	// 		i++
	// 	}

	// 	if y == s {
	// 		count++
	// 	}

	// 	if y != 0 && y.stack(x) != s {
	// 		count2++
	// 	}
	// }

	// fmt.Println("count:", count)
	// fmt.Println("count2:", count2)

	// findAllPossibleShapes()
}

func (s shape) isPossible() bool {
	_, ok := slices.BinarySearch(possibleList, s)
	return ok
}

func (s shape) unstack() (bottom, top shape) {
	if s == 0 {
		return 0, 0
	}
	mask := shape(0b1111_0000_0000_0000_1111) << ((s.layerCount() - 1) * 4)
	return s &^ mask, s & mask
}
func (s shape) unstackBottom() (bottom, top shape) {
	mask := shape(0b1111_0000_0000_0000_1111)
	return s & mask, (s &^ mask) >> 4
}

func readPossibleShapes() {
	file, _ := os.Open("possible-sorted.bin")
	defer file.Close()

	var buffer bytes.Buffer
	io.Copy(&buffer, file)

	bytes := buffer.Bytes()
	possibleList = *(*[]shape)(unsafe.Pointer(&bytes))
	possibleList = possibleList[: len(bytes)/4 : len(bytes)/4]

	// var b [4]byte
	// for {
	// 	_, err := buffer.Read(b[:])
	// 	if err != nil {
	// 		break
	// 	}
	// 	s := shape(binary.LittleEndian.Uint32(b[:]))
	// 	possible[s] = struct{}{}
	// 	possibleList = append(possibleList, s)
	// }
}

func findAllPossibleShapes() {
	for i := range additions {
		additions[i] = make([]shape, 0, 400_000_000)
	}

	// fmt.Println(shapeFrom("cu------:cu------").pushPins().pushPins().pushPins())

	addPossible(shapeFrom("Cu------"))
	addPossible(shapeFrom("P-------"))

	for changed {
		changed = false
		for _, shape := range possibleList {
			addPossible(shape.rotate())
			addPossible(shape.pushPins())
			addPossible(shape.right())
		}

		for _, a := range possibleLeftList {
			for _, b := range possibleRightList {
				addPossible(a.combine(b))
			}
		}
	}

	for _, shape := range possibleList {
		if shape.layerCount() == 1 {
			stackables = append(stackables, shape)
		}
	}
	slices.Sort(stackables)

	changed = true
	for changed {
		changed = false

		changed = true
		for changed {
			changed = false

			fmt.Println("Simple...", len(possibleList))
			var wg sync.WaitGroup
			wg.Add(len(additions))
			for i := range additions {
				go calcSimple(i, &wg)
			}
			wg.Wait()
			fmt.Println("Merging Simple...", len(possibleList))
			for i := range additions {
				for _, shape := range additions[i] {
					addPossible(shape)
				}
			}
		}

		fmt.Println("Combine...", len(possibleList))
		changed = true
		for changed {
			changed = false
			for _, a := range possibleLeftList {
				for _, b := range possibleRightList {
					addPossible(a.combine(b))
				}
			}
		}

		fmt.Println("Stack...", len(possibleList))
		var wg sync.WaitGroup
		wg.Add(len(additions))
		for i := range additions {
			go calcStack(i, &wg)
		}
		wg.Wait()
		fmt.Println("Merging Stack...", len(possibleList))
		for i := range additions {
			for _, shape := range additions[i] {
				addPossible(shape)
			}
		}

		fmt.Println("Simple (1)...", len(possibleList))
		for _, shape := range possibleList {
			addPossible(shape.rotate())
			addPossible(shape.pushPins())
			addPossible(shape.right())
			addPossible(shape.crystalGenerator())
		}

		fmt.Println("Combine (1)...", len(possibleList))
		for _, a := range possibleLeftList {
			for _, b := range possibleRightList {
				addPossible(a.combine(b))
			}
		}
	}

	fmt.Println(len(possible)) //41166043

	slices.Sort(possibleList)

	pointer := unsafe.SliceData(possibleList)
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(pointer)), len(possibleList)*4)
	file, _ := os.Create("possible-sorted.bin")
	defer file.Close()
	file.Write(bytes)
	// file, _ := os.Create("possible.bin")
	// defer file.Close()

	// for _, shape := range possibleList {
	// 	var b [4]byte
	// 	binary.LittleEndian.PutUint32(b[:], uint32(shape))
	// 	file.Write(b[:])
	// }
}

func calcStack(index int, wg *sync.WaitGroup) {
	result := additions[index][:0]
	for _, b := range stackables[len(stackables)/len(additions)*index : len(stackables)/len(additions)*(index+1)] {
		for _, a := range possibleList {
			c := a.stack(b)
			if c != a {
				if _, ok := possible[c]; !ok {
					result = append(result, c)
				}
			}
		}
	}
	additions[index] = result
	wg.Done()
}

func calcSimple(index int, wg *sync.WaitGroup) {
	max := len(possibleList) / len(additions) * (index + 1)
	if index == len(additions)-1 {
		max = len(possibleList)
	}
	result := additions[index][:0]
	for _, shape := range possibleList[len(possibleList)/len(additions)*index : max] {
		s := shape.rotate()
		if _, ok := possible[s]; !ok {
			result = append(result, s)
		}
		s = shape.rotate().rotate()
		if _, ok := possible[s]; !ok {
			result = append(result, s)
		}
		s = shape.pushPins()
		if _, ok := possible[s]; !ok {
			result = append(result, s)
		}
		s = shape.right()
		if _, ok := possible[s]; !ok {
			result = append(result, s)
		}
		s = shape.crystalGenerator()
		if _, ok := possible[s]; !ok {
			result = append(result, s)
		}
	}
	additions[index] = result
	wg.Done()
}
