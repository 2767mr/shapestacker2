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
	possibleRecipes   []recipe

	additions  = make([][]recipe, 8)
	stackables = []shape{}

	changed = true
)

func addPossible(s shape, source shape) {
	if _, ok := possible[s]; !ok {
		possible[s] = struct{}{}
		possibleList = append(possibleList, s)
		possibleRecipes = append(possibleRecipes, recipe{s, source})
		changed = true

		if s&0b1100_1100_1100_1100_1100_1100_1100_1100 == 0 {
			possibleLeftList = append(possibleLeftList, s)
		}
		if s&^0b1100_1100_1100_1100_1100_1100_1100_1100 == 0 {
			possibleRightList = append(possibleRightList, s)
		}
	}
}

type recipe struct {
	shape    shape
	original shape
}

func main() {
	if _, err := os.Stat("./possible-sorted.bin"); os.IsNotExist(err) {
		findAllPossibleShapes()
	} else {
		readPossibleShapes()
	}

	fmt.Println("Possible shapes:", len(possibleList), len(possibleRecipes))
	for i := range possibleList {
		if possibleList[i] != possibleRecipes[i].shape {
			fmt.Println("Mismatch:", possibleList[i], possibleRecipes[i].shape)
		}
	}

	// x := readFromFatcatX()
	// fmt.Println("Possible shapes (fatcatx):", len(x))

	// slices.Sort(x)

	// count := 0
	// for _, s := range possibleList {
	// 	count++
	// 	if _, ok := slices.BinarySearch(x, s); !ok {
	// 		r, _ := slices.BinarySearchFunc(possibleRecipes, s, func(a recipe, b shape) int {
	// 			return int(a.shape) - int(b)
	// 		})
	// 		fmt.Println("Missing:", s, possibleRecipes[r].original)
	// 	}
	// }
	// fmt.Println("Count:", count)

	printRecipe(shapeFrom("----CuCu:----Cu--:----P-cu:----Cucu"))
	fmt.Println("---")
	printRecipe(shapeFrom("CuCu----:--P-----:cuCu----:cu------"))
	fmt.Println("---")
	printRecipe(shapeFrom("cuCu----:CuP-----:cuCu----:cu------"))

	// base := shapeFrom("----CuP-:CuCucuP-")
	// for _, s := range possibleList {
	// 	if s&0b1111_1111_0000_0000_1111_1111 == base {
	// 		fmt.Println(s)
	// 	}
	// }

	// fmt.Println(shapeFrom("P-------:Rg------:cbRb--Rb").isPossible())

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

func printRecipe(s shape) {
	stackables = []shape{
		shapeFrom("P-------"),
		shapeFrom("--P-----"),
		shapeFrom("----P---"),
		shapeFrom("------P-"),
		shapeFrom("Cu------"),
		shapeFrom("--Cu----"),
		shapeFrom("----Cu--"),
		shapeFrom("------Cu"),
		shapeFrom("CuCu----"),
		shapeFrom("--CuCu--"),
		shapeFrom("----CuCu"),
		shapeFrom("Cu----Cu"),
		shapeFrom("CuCuCu--"),
		shapeFrom("--CuCuCu"),
		shapeFrom("Cu--CuCu"),
		shapeFrom("CuCu--Cu"),
		shapeFrom("CuCuCuCu"),
	}

	visited := make(map[shape]struct{})
	var ok bool
	for ; !ok && s != 0; _, ok = visited[s] {
		visited[s] = struct{}{}
		i, _ := slices.BinarySearchFunc(possibleRecipes, s, func(a recipe, b shape) int {
			return int(a.shape) - int(b)
		})

		r := possibleRecipes[i]

		op := "unknown"
		if r.original.rotate() == s {
			op = "rotate"
		} else if r.original.rotate().rotate() == s {
			op = "rotate2"
		} else if r.original.pushPins() == s {
			op = "pushPins"
		} else if r.original.right() == s {
			op = "right"
		} else if r.original.crystalGenerator() == s {
			op = "crystalGenerator"
		} else {
			for _, stackable := range stackables {
				if r.original.stack(stackable) == s {
					op = "stack " + stackable.String()
					break
				}
			}

			if op == "unknown" {
				if r.original&0b1100_1100_1100_1100_1100_1100_1100_1100 == 0 && (s & 0b1100_1100_1100_1100_1100_1100_1100_1100).isPossible() && (s&0b1100_1100_1100_1100_1100_1100_1100_1100) != 0 {
					op = "combine " + (s & 0b1100_1100_1100_1100_1100_1100_1100_1100).String()
				} else if r.original&^0b1100_1100_1100_1100_1100_1100_1100_1100 == 0 && (s &^ 0b1100_1100_1100_1100_1100_1100_1100_1100).isPossible() && (s&^0b1100_1100_1100_1100_1100_1100_1100_1100) != 0 {
					op = "combine " + (s &^ 0b1100_1100_1100_1100_1100_1100_1100_1100).String()
				}
			}
		}

		fmt.Println("Recipe:", s, "<-", r.original, op)
		s = r.original
	}
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

	bbytes := buffer.Bytes()
	possibleList = *(*[]shape)(unsafe.Pointer(&bbytes))
	possibleList = possibleList[: len(bbytes)/4 : len(bbytes)/4]

	file2, _ := os.Open("possible-sorted-recipe.bin")
	defer file2.Close()

	var buffer2 bytes.Buffer
	io.Copy(&buffer2, file2)

	bytes2 := buffer2.Bytes()
	possibleRecipes = *(*[]recipe)(unsafe.Pointer(&bytes2))
	possibleRecipes = possibleRecipes[: len(bytes2)/8 : len(bytes2)/8]

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
		additions[i] = make([]recipe, 0, 400_000_000)
	}

	// fmt.Println(shapeFrom("cu------:cu------").pushPins().pushPins().pushPins())

	stackables = []shape{
		shapeFrom("P-------"),
		shapeFrom("--P-----"),
		shapeFrom("----P---"),
		shapeFrom("------P-"),
		shapeFrom("Cu------"),
		shapeFrom("--Cu----"),
		shapeFrom("----Cu--"),
		shapeFrom("------Cu"),
		shapeFrom("CuCu----"),
		shapeFrom("--CuCu--"),
		shapeFrom("----CuCu"),
		shapeFrom("Cu----Cu"),
		shapeFrom("CuCuCu--"),
		shapeFrom("--CuCuCu"),
		shapeFrom("Cu--CuCu"),
		shapeFrom("CuCu--Cu"),
		shapeFrom("CuCuCuCu"),
	}

	for _, shape := range stackables {
		addPossible(shape, 0)
	}

	// for changed {
	// 	changed = false
	// 	for _, shape := range possibleList {
	// 		addPossible(shape.rotate())
	// 		addPossible(shape.pushPins())
	// 		addPossible(shape.right())
	// 	}

	// 	for _, a := range possibleLeftList {
	// 		for _, b := range possibleRightList {
	// 			addPossible(a.combine(b))
	// 		}
	// 	}
	// }

	// for _, shape := range possibleList {
	// 	if shape.layerCount() == 1 {
	// 		stackables = append(stackables, shape)
	// 	}
	// }
	slices.Sort(stackables)

	changed = true
	for changed {
		changed = false

		changed = true
		for changed {
			changed = false

			fmt.Println("Simple...", len(possibleList))
			slices.Sort(possibleList)
			var wg sync.WaitGroup
			wg.Add(len(additions))
			for i := range additions {
				go calcSimple(i, &wg)
			}
			wg.Wait()
			fmt.Println("Merging Simple...", len(possibleList))
			for i := range additions {
				for _, shape := range additions[i] {
					addPossible(shape.shape, shape.original)
				}
			}
		}

		fmt.Println("Combine...", len(possibleList))
		changed = true
		for changed {
			changed = false
			for _, a := range possibleLeftList {
				for _, b := range possibleRightList {
					addPossible(a.combine(b), a)
				}
			}
		}

		fmt.Println("Stack...", len(possibleList))
		slices.Sort(possibleList)
		var wg sync.WaitGroup
		wg.Add(len(additions))
		for i := range additions {
			go calcStack(i, &wg)
		}
		wg.Wait()
		fmt.Println("Merging Stack...", len(possibleList))
		for i := range additions {
			for _, shape := range additions[i] {
				addPossible(shape.shape, shape.original)
			}
		}

		fmt.Println("Simple (1)...", len(possibleList))
		slices.Sort(possibleList)
		wg.Add(len(additions))
		for i := range additions {
			go calcSimple(i, &wg)
		}
		wg.Wait()
		fmt.Println("Merging Simple (1)...", len(possibleList))
		for i := range additions {
			for _, shape := range additions[i] {
				addPossible(shape.shape, shape.original)
			}
		}

		fmt.Println("Combine (1)...", len(possibleList))
		for _, a := range possibleLeftList {
			for _, b := range possibleRightList {
				addPossible(a.combine(b), a)
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

	slices.SortFunc(possibleRecipes, func(a, b recipe) int {
		return int(a.shape) - int(b.shape)
	})
	pointer2 := unsafe.SliceData(possibleRecipes)
	bytes = unsafe.Slice((*byte)(unsafe.Pointer(pointer2)), len(possibleRecipes)*8)
	file2, _ := os.Create("possible-sorted-recipe.bin")
	defer file2.Close()
	file2.Write(bytes)
	// file, _ := os.Create("possible.bin")
	// defer file.Close()

	// for _, shape := range possibleList {
	// 	var b [4]byte
	// 	binary.LittleEndian.PutUint32(b[:], uint32(shape))
	// 	file.Write(b[:])
	// }
}

func calcStack(index int, wg *sync.WaitGroup) {
	max := len(stackables) / len(additions) * (index + 1)
	if index == len(stackables)-1 {
		max = len(stackables)
	}
	result := additions[index][:0]
	for _, b := range stackables[len(stackables)/len(additions)*index : max] {
		for _, a := range possibleList {
			c := a.unsafeStack(b)
			if c != a {
				if !c.isPossible() {
					result = append(result, recipe{c, a})
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
		if !s.isPossible() {
			result = append(result, recipe{s, shape})
		}
		s = shape.rotate().rotate()
		if !s.isPossible() {
			result = append(result, recipe{s, shape})
		}
		s = shape.pushPins()
		if !s.isPossible() {
			result = append(result, recipe{s, shape})
		}
		s = shape.right()
		if !s.isPossible() {
			result = append(result, recipe{s, shape})
		}
		s = shape.crystalGenerator()
		if !s.isPossible() {
			result = append(result, recipe{s, shape})
		}
	}
	additions[index] = result
	wg.Done()
}
