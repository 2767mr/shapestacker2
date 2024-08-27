package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"sync"
	"unsafe"
)

var (
	possible = make(map[Shape]struct{}) //400_000_000

	possibleList      []Shape
	possibleLeftList  []Shape
	possibleRightList []Shape
	possibleRecipes   []recipe

	reverseRight   = make(map[Shape]Shape, 400_000_000)
	reversePinPush = make(map[Shape]Shape, 400_000_000)

	hardcodedPins   = make(map[Shape]Shape)
	hardcodedStacks = make(map[Shape]Shape)

	additions  = make([][]recipe, 8)
	stackables = []Shape{
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

	recipeMap = make(map[Shape]recipeNode)

	changed = true
)

func addPossible(s Shape, source Shape) {
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
	shape    Shape
	original Shape
}

type recipeNode struct {
	shape     Shape
	operation string
	original1 Shape
	original2 Shape
}

func makeRecipe(s Shape) recipeNode {
	if s.toFilled() <= 0b1111 {
		return recipeNode{
			shape:     s,
			operation: "trivial",
		}
	}

	if !s.topLayer().hasCrystal() {
		bottom, top := s.unstack()
		return recipeNode{
			operation: "stack",
			shape:     s,
			original1: bottom,
			original2: top >> ((s.layerCount() - 1) * 4),
		}
	}

	if s&^0b0011_0011_0011_0011_0011_0011_0011_0011 == 0 {
		return recipeNode{shape: s, operation: "half"}
	}

	if s.isLeftRightValid() {
		right := (s & 0b0011_0011_0011_0011_0011_0011_0011_0011)
		left := (s &^ 0b0011_0011_0011_0011_0011_0011_0011_0011)
		if left.collapse() == left && right.collapse() == right {
			return recipeNode{shape: s, operation: "combine", original1: left, original2: right}
		}
	}

	if s.isUpDownValid() {
		//rotate then do combine via isLeftRightValid
		return recipeNode{shape: s, operation: "rotate", original1: s.rotate().rotate().rotate()}
	}

	if !s.isMinimal() {
		return recipeNode{shape: s, operation: "rotate", original1: s.rotate().rotate().rotate()}
	}

	if s.isStackTopWithoutCrystals() {
		bottom, top := s.unstack()
		bottom |= top.crystals()
		return recipeNode{
			operation: "stack",
			shape:     s,
			original1: bottom,
			original2: (top &^ bottom).removeBottomEmpty(),
		}
	}

	r := s.recipe()
	if r.original.pushPins() == s {
		hardcodedPins[s] = r.original
		return recipeNode{shape: s, operation: "pushPins", original1: r.original}
	}

	if r.original.stack((s &^ r.original).removeBottomEmpty()) == s {
		hardcodedStacks[s] = r.original
		return recipeNode{shape: s, operation: "stack", original1: r.original, original2: (s &^ r.original).removeBottomEmpty()}
	}

	if r.original.rotate() == s {
		r2 := r.original.recipe()
		if r2.original.rotate().pushPins() == s {
			hardcodedPins[s] = r2.original.rotate()
			return recipeNode{shape: s, operation: "pushPins", original1: r2.original.rotate()}
		}

		if r2.original.rotate().stack((s &^ r2.original.rotate()).removeBottomEmpty()) == s {
			hardcodedStacks[s] = r2.original.rotate()
			return recipeNode{shape: s, operation: "stack", original1: r2.original.rotate(), original2: (s &^ r2.original.rotate()).removeBottomEmpty()}
		}
	}

	if p, ok := reversePinPush[s]; ok {
		hardcodedPins[s] = p
		return recipeNode{shape: s, operation: "pushPins", original1: p}
	}

	// for i := position(16); i > 0; i-- {
	// 	if !s.cornerAt(i - 1).isCrystal() {
	// 		group := s.connectedGroup(i-1, 0)
	// 		if (s &^ group).stack(group.removeBottomEmpty()) == s {
	// 			return recipeNode{shape: s, operation: "stack", original1: s &^ group, original2: group.removeBottomEmpty()}
	// 		}
	// 	}
	// }

	return recipeNode{shape: s, operation: "unknown"}
}

func (r recipeNode) check() bool {
	switch r.operation {
	case "unknown":
		return false //Temporary
	case "trivial":
		return r.shape.isPossible() && r.shape.toFilled() <= 0b1111
	case "half":
		return r.shape.isPossible() && r.shape&^0b0011_0011_0011_0011_0011_0011_0011_0011 == 0
	case "stack":
		return r.original1.stack(r.original2) == r.shape && r.original1.isPossible() && r.original2.isPossible()
	case "rotate":
		return r.original1.rotate() == r.shape && r.original1.isPossible()
	case "combine":
		return r.original1.combine(r.original2) == r.shape && r.original1.isPossible() && r.original2.isPossible()
	case "pushPins":
		return r.original1.pushPins() == r.shape && r.original1.isPossible()
	default:
		return false
	}
}

func main() {
	if _, err := os.Stat("./possible-sorted.bin"); os.IsNotExist(err) {
		findAllPossibleShapes()
	} else {
		readPossibleShapes()
	}

	fmt.Println("Possible shapes:", len(possibleList)-1)
	// for i := range possibleList {
	// 	if possibleList[i] != possibleRecipes[i].shape {
	// 		fmt.Println("Mismatch:", possibleList[i], possibleRecipes[i].shape)
	// 	}
	// }

	for _, s := range possibleList {
		reversePinPush[s.pushPins()] = s
		// reverseRight[s.right()] = s
	}

	fmt.Println("finding recipes")

	// printRecipe(shapeFrom("--P---P-:--Cu--cu:----CuCu:Cu--Cucu"))
	// fmt.Println("---")
	// printRecipe(shapeFrom("--cu----:Cucu----:--cu----"))
	// fmt.Println("---")
	// printRecipe(shapeFrom("cuCu----:CuP-----:cuCu----:cu------"))
	// return

	count := 0
	for _, s := range possibleList[1:] {
		r := makeRecipe(s)
		if !r.check() {
			count++
			if count < 1000 {
				fmt.Println("Error:", s, r.operation, r.original1, r.original2, "|", r.shape.recipe().original, findRecipeFromPreloaded(s).operation)
			}

		}
	}
	fmt.Println("Invalid recipes:", count)

	fPins, _ := os.Create("hardcoded-pins.json")
	defer fPins.Close()
	json.NewEncoder(fPins).Encode(hardcodedPins)

	fStacks, _ := os.Create("hardcoded-stacks.json")
	defer fStacks.Close()
	json.NewEncoder(fStacks).Encode(hardcodedStacks)

	return

	// base := shapeFrom("--Cu--Cu:Cu--Cucu")

	// count := 0
	// var reduced []shape
	for _, s := range possibleList[1:] {
		if s.isMinimal() && s.topLayer().hasCrystal() && !s.isTrivialPinPusher() && !(s.isUpDownValid() || s.isLeftRightValid()) && !s.isStackTopWithoutCrystals() { //s.toFilled()&^0b0011_0011_0011_0011 == 0
			// if base == s {
			// 	fmt.Println("Filling recipe:", s)
			// }
			// fmt.Println(s)
			count++
			fillRecipe(s)

			// reduced = append(reduced, s)
		}
	}
	for _, s := range possibleList[1:] {
		if s.isMinimal() && s.topLayer().hasCrystal() && !s.isTrivialPinPusher() && s.toFilled()&^0b0011_0011_0011_0011 == 0 {
			// fmt.Println(s)
			count++
			fillRecipe(s)
			// reduced = append(reduced, s)
		}
	}
	fmt.Println("map:", len(recipeMap))

	checkRecipeMap()

	// p1 := base.isPossible()
	// p2 := base.rotate().isPossible()
	// p3 := base.rotate().rotate().isPossible()
	// p4 := base.rotate().rotate().rotate().isPossible()

	// _, _, _, _ = p1, p2, p3, p4
	// findRoute(base)

	for _, s := range possibleList[1:] {
		findRoute(s)
	}

	// pointer := unsafe.SliceData(reduced)
	// bytes := unsafe.Slice((*byte)(unsafe.Pointer(pointer)), len(reduced)*4)
	// file, _ := os.Create("possible-reduced.bin")
	// defer file.Close()
	// file.Write(bytes)
	// runtime.KeepAlive(reduced)

	// count = 0
	// for _, s := range possibleList[1:] {
	// 	if !s.topLayer().hasCrystal() {
	// 		bottom, _ := s.unstack()
	// 		if !bottom.isPossible() {
	// 			fmt.Println("Unstack:", s)
	// 		}
	// 		count++
	// 	}
	// }
	// fmt.Println("top:", count)

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

func findRecipe(s Shape) recipeNode {
	return findRecipeFromPreloaded(s)
	if s.toFilled() <= 0b1111_1111_11111 {
		return findRecipeFromPreloaded(s)
	}

	if !s.topLayer().hasCrystal() {
		b, _ := s.unstack()
		return recipeNode{
			operation: "stack",
			shape:     s,
			original1: b,
		}
	}

	if s.isStackTopWithoutCrystals() {
		b, t := s.unstack()
		return recipeNode{
			operation: "stack",
			shape:     s,
			original1: b | t.crystals(),
		}
	}

	if s.isLeftRightValid() {
		right := (s & 0b0011_0011_0011_0011_0011_0011_0011_0011).collapse()
		left := (s &^ 0b0011_0011_0011_0011_0011_0011_0011_0011).collapse()
		if left.isPossible() && right.isPossible() {
			return recipeNode{
				operation: "combine",
				shape:     s,
				original1: s & 0b0011_0011_0011_0011_0011_0011_0011_0011,
				original2: s &^ 0b0011_0011_0011_0011_0011_0011_0011_0011,
			}
		}
	}

	if p, ok := reversePinPush[s]; ok {
		return recipeNode{
			operation: "pin",
			shape:     s,
			original1: p,
		}
	}

	if r, ok := reverseRight[s]; ok {
		return recipeNode{
			operation: "right",
			shape:     s,
			original1: r,
		}
	}

	return recipeNode{
		operation: "rotate",
		shape:     s,
		original1: s.rotate().rotate().rotate(),
	}
}

func findRecipeDepth(original Shape, s Shape, depth int) int {
	if depth > 100 {
		fmt.Println("Circle:", original, s)
		return 0
	}
	if s == 0 {
		return 0
	}

	if s.layerCount() == 1 {
		return depth
	}

	r := findRecipe(s)
	return max(findRecipeDepth(original, r.original1, depth+1), findRecipeDepth(original, r.original2, depth+1))
}

func checkRecipeMap() {
	for s := range recipeMap {
		stack := []Shape{s}

		for len(stack) > 0 {
			r := recipeMap[stack[len(stack)-1]]

			if r.original1.toFilled()&^0b1111 != 0 {
				if r.original1 == s {
					fmt.Println("Error recipe:", s)
					return
				}
				stack = append(stack, r.original1)
			}
			if r.original2.toFilled()&^0b1111 != 0 {
				if r.original2 == s {
					fmt.Println("Error recipe:", s)
					return
				}
				stack = append(stack, r.original2)
			}
		}

	}
}

func findRoute(s Shape) {
	if s.toFilled() <= 0b1111 {
		return
	}

	s = s.minimal()

	r, ok := recipeMap[s]
	if ok {
		rStack := []recipeNode{r}

		for len(rStack) > 0 {
			if len(rStack) > 20000 {
				fmt.Println("Stack too big:", s)
				return
			}

			r := rStack[len(rStack)-1]
			rStack = rStack[:len(rStack)-1]

			if r.original1.toFilled()&^0b1111 != 0 {
				newR, ok := recipeMap[r.original1]
				if !ok {
					fmt.Println("No recipe:", s, r.shape)
					return
				}
				rStack = append(rStack, newR)
			}

			if r.original2.toFilled()&^0b1111 != 0 {
				newR, ok := recipeMap[r.original2]
				if !ok {
					fmt.Println("No recipe:", s, r.shape)
					return
				}
				rStack = append(rStack, newR)
			}
		}
		return
	}

	if !s.topLayer().hasCrystal() {
		bottom, top := s.unstack()
		findRoute(bottom)
		if bottom.stack(top) != s {
			fmt.Println("Error stack:", s)
		}
		return
	}

	if s.isTrivialPinPusher() {
		bottom, top := s.unstackBottom()
		findRoute(top)
		if !bottom.isPins() || top.pushPins() != s {
			fmt.Println("Error pushPins:", s)
		}
		return
	}

	if s.isLeftRightValid() {
		right := (s & 0b0011_0011_0011_0011_0011_0011_0011_0011).collapse()
		left := (s &^ 0b0011_0011_0011_0011_0011_0011_0011_0011).collapse()
		if left != 0 && right != 0 {
			findRoute(left)
			findRoute(right)
			if left.combine(right) != s {
				fmt.Println("Error combine lr:", s)
			}
			return
		}
	}

	if s.isUpDownValid() {
		up := (s &^ 0b0110_0110_0110_0110_0110_0110_0110_0110).collapse()
		down := (s & 0b0110_0110_0110_0110_0110_0110_0110_0110).collapse()
		if up != 0 && down != 0 {
			findRoute(up)
			findRoute(down)
			if up.combine(down) != s {
				fmt.Println("Error combine ud:", s)
			}
			return
		}
	}

	if s.isStackTopWithoutCrystals() {
		bottom, top := s.unstack()
		bottom |= top.crystals()
		findRoute(bottom)
		if bottom.stack(top&^bottom) != s {
			fmt.Println("Error stack top no crystals:", s)
		}
		return
	}

	fmt.Println("No recipe:", s)
}

func fillRecipe(s Shape) {
	if _, ok := recipeMap[s]; ok || s == 0 {
		return
	}

	result := findRecipeFromPreloaded(s)
	recipeMap[s] = result

	fillRecipe(result.original1)
	fillRecipe(result.original2)
}

func findRecipeFromPreloaded(s Shape) recipeNode {
	if _, ok := recipeMap[s]; ok || s == 0 {
		return recipeNode{operation: "unknown", shape: s}
	}

	i, _ := slices.BinarySearchFunc(possibleRecipes, s, func(a recipe, b Shape) int {
		return int(a.shape) - int(b)
	})

	r := possibleRecipes[i]

	result := recipeNode{shape: s, original1: r.original}

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
				op = "stack"
				result.original2 = stackable
				break
			}
		}

		if op == "unknown" {
			if r.original&0b1100_1100_1100_1100_1100_1100_1100_1100 == 0 && (s & 0b1100_1100_1100_1100_1100_1100_1100_1100).isPossible() && (s&0b1100_1100_1100_1100_1100_1100_1100_1100) != 0 {
				op = "combine"
				result.original2 = s & 0b1100_1100_1100_1100_1100_1100_1100_1100

			} else if r.original&^0b1100_1100_1100_1100_1100_1100_1100_1100 == 0 && (s &^ 0b1100_1100_1100_1100_1100_1100_1100_1100).isPossible() && (s&^0b1100_1100_1100_1100_1100_1100_1100_1100) != 0 {
				op = "combine"
				result.original2 = s &^ 0b1100_1100_1100_1100_1100_1100_1100_1100
			}
		}
	}

	if op == "unknown" {
		fmt.Println("Unknown operation:", s)
		return recipeNode{operation: "unknown", shape: s}
	}

	result.operation = op
	return result
}

func printRecipe(s Shape) {
	stackables = []Shape{
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

	visited := make(map[Shape]struct{})
	var ok bool
	for ; !ok && s != 0; _, ok = visited[s] {
		visited[s] = struct{}{}
		i, _ := slices.BinarySearchFunc(possibleRecipes, s, func(a recipe, b Shape) int {
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

func (s Shape) isPossible() bool {
	_, ok := slices.BinarySearch(possibleList, s)
	return ok
}

func (s Shape) unstack() (bottom, top Shape) {
	if s == 0 {
		return 0, 0
	}
	mask := Shape(0b1111_0000_0000_0000_1111) << ((s.layerCount() - 1) * 4)
	return s &^ mask, s & mask
}
func (s Shape) unstackBottom() (bottom, top Shape) {
	mask := Shape(0b1111_0000_0000_0000_1111)
	return s & mask, (s &^ mask) >> 4
}

func readPossibleShapes() {
	file, _ := os.Open("possible-sorted.bin")
	defer file.Close()

	var buffer bytes.Buffer
	io.Copy(&buffer, file)

	bbytes := buffer.Bytes()
	possibleList = *(*[]Shape)(unsafe.Pointer(&bbytes))
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

	stackables = []Shape{
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
