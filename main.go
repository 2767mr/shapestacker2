package main

import (
	"bytes"
	"encoding/binary"
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

	recipeMap = make(map[Shape]RecipeNode)

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

type RecipeNode struct {
	Shape     Shape  `json:"-"`
	Operation string `json:"operation"`
	Original1 Shape  `json:"original1,omitempty"`
	Original2 Shape  `json:"original2,omitempty"`
}

func makeRecipe(s Shape) RecipeNode {
	if s.toFilled() <= 0b1111 {
		return RecipeNode{
			Shape:     s,
			Operation: "trivial",
		}
	}

	if !s.topLayer().hasCrystal() {
		bottom, top := s.unstack()
		return RecipeNode{
			Operation: "stack",
			Shape:     s,
			Original1: bottom,
			Original2: top >> ((s.layerCount() - 1) * 4),
		}
	}

	if s&^0b0011_0011_0011_0011_0011_0011_0011_0011 == 0 {
		return RecipeNode{Shape: s, Operation: "half"}
	}

	if s.isLeftRightValid() {
		right := (s & 0b0011_0011_0011_0011_0011_0011_0011_0011)
		left := (s &^ 0b0011_0011_0011_0011_0011_0011_0011_0011)
		if left.collapse() == left && right.collapse() == right {
			return RecipeNode{Shape: s, Operation: "combine", Original1: left, Original2: right}
		}
	}

	if s.isUpDownValid() {
		//rotate then do combine via isLeftRightValid
		return RecipeNode{Shape: s, Operation: "rotate", Original1: s.rotate().rotate().rotate()}
	}

	if !s.isMinimal() {
		if s.rotate().isMinimal() || s.rotate().rotate().isMinimal() || s.rotate().rotate().rotate().isMinimal() {
			return RecipeNode{Shape: s, Operation: "rotate", Original1: s.rotate().rotate().rotate()}
		} else {
			return RecipeNode{Shape: s, Operation: "mirror", Original1: s.mirror()}
		}
	}

	if s.isStackTopWithoutCrystals() {
		bottom, top := s.unstack()
		bottom |= top.crystals()
		return RecipeNode{
			Operation: "stack",
			Shape:     s,
			Original1: bottom,
			Original2: (top &^ bottom).removeBottomEmpty(),
		}
	}

	r := s.recipe()
	if r.original.pushPins() == s {
		hardcodedPins[s] = r.original
		return RecipeNode{Shape: s, Operation: "pushPins", Original1: r.original}
	}

	if r.original.stack((s &^ r.original).removeBottomEmpty()) == s {
		hardcodedStacks[s] = r.original
		return RecipeNode{Shape: s, Operation: "stack", Original1: r.original, Original2: (s &^ r.original).removeBottomEmpty()}
	}

	if r.original.rotate() == s {
		r2 := r.original.recipe()
		if r2.original.rotate().pushPins() == s {
			hardcodedPins[s] = r2.original.rotate()
			return RecipeNode{Shape: s, Operation: "pushPins", Original1: r2.original.rotate()}
		}

		if r2.original.rotate().stack((s &^ r2.original.rotate()).removeBottomEmpty()) == s {
			hardcodedStacks[s] = r2.original.rotate()
			return RecipeNode{Shape: s, Operation: "stack", Original1: r2.original.rotate(), Original2: (s &^ r2.original.rotate()).removeBottomEmpty()}
		}
	}

	if p, ok := reversePinPush[s]; ok {
		hardcodedPins[s] = p
		return RecipeNode{Shape: s, Operation: "pushPins", Original1: p}
	}

	// for i := position(16); i > 0; i-- {
	// 	if !s.cornerAt(i - 1).isCrystal() {
	// 		group := s.connectedGroup(i-1, 0)
	// 		if (s &^ group).stack(group.removeBottomEmpty()) == s {
	// 			return recipeNode{shape: s, operation: "stack", original1: s &^ group, original2: group.removeBottomEmpty()}
	// 		}
	// 	}
	// }

	return RecipeNode{Shape: s, Operation: "unknown"}
}

func (r RecipeNode) check() bool {
	switch r.Operation {
	case "unknown":
		return false //Temporary
	case "trivial":
		return r.Shape.isPossible() && r.Shape.toFilled() <= 0b1111
	case "half":
		return r.Shape.isPossible() && r.Shape&^0b0011_0011_0011_0011_0011_0011_0011_0011 == 0
	case "stack":
		return r.Original1.stack(r.Original2) == r.Shape && r.Original1.isPossible() && r.Original2.isPossible()
	case "rotate":
		return r.Original1.rotate() == r.Shape && r.Original1.isPossible()
	case "mirror":
		return r.Original1.mirror() == r.Shape && r.Original1.isPossible()
	case "combine":
		return r.Original1.combine(r.Original2) == r.Shape && r.Original1.isPossible() && r.Original2.isPossible()
	case "pushPins":
		return r.Original1.pushPins() == r.Shape && r.Original1.isPossible()
	default:
		return false
	}
}

func (r RecipeNode) WriteTo(w io.Writer) error {
	switch r.Operation {
	case "unknown":
		w.Write([]byte{'u'})
	case "trivial":
		w.Write([]byte{'t'})
	case "half":
		w.Write([]byte{'h'})
	case "stack":
		w.Write([]byte{'s'})
	case "rotate":
		w.Write([]byte{'r'})
	case "combine":
		w.Write([]byte{'c'})
	case "pushPins":
		w.Write([]byte{'p'})
	default:
		w.Write([]byte{'u'})
	}

	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(r.Shape))
	w.Write(b[:])

	binary.LittleEndian.PutUint32(b[:], uint32(r.Original1))
	w.Write(b[:])

	binary.LittleEndian.PutUint32(b[:], uint32(r.Original2))
	_, err := w.Write(b[:])
	return err
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

	// for _, s := range possibleList {
	// 	reversePinPush[s.pushPins()] = s
	// 	// reverseRight[s.right()] = s
	// }

	fmt.Println("finding recipes")

	// makeRecipe(shapeFrom("cu----P-:CuCuCuP-:------cu"))

	// printRecipe2(shapeFrom("----CuCu:----Cu--:----P-cr:----Cucr"))
	// return

	// printRecipe(shapeFrom("--P---P-:--Cu--cu:----CuCu:Cu--Cucu"))
	// fmt.Println("---")
	// printRecipe(shapeFrom("--cu----:Cucu----:--cu----"))
	// fmt.Println("---")
	// printRecipe(shapeFrom("cuCu----:CuP-----:cuCu----:cu------"))
	// return

	// var buf bytes.Buffer

	count := 0
	for _, s := range possibleList[1:] {
		if s.toFilled() > 0b1111 && s.topLayer().hasCrystal() && s&^0b0011_0011_0011_0011_0011_0011_0011_0011 == 0 {
			findRecipeDepth(s, s, 0)
			// mirrored := s.mirror()
			// otherSide := mirrored &^ (mirrored.crystals() &^ 0b1111_1111_1111_1111)
			// combined := s.combine(otherSide)

			// emptySpaces := (combined.toFilled() >> 4) &^ combined.toFilled()
			// combined |= emptySpaces | (emptySpaces << 16)

			// emptySpaces = (combined.toFilled() >> 4) &^ combined.toFilled()
			// combined |= emptySpaces | (emptySpaces << 16)

			// emptySpaces = (combined.toFilled() >> 4) &^ combined.toFilled()
			// combined |= emptySpaces | (emptySpaces << 16)

			// up := combined &^ 0b0110_0110_0110_0110_0110_0110_0110_0110
			// down := combined & 0b0110_0110_0110_0110_0110_0110_0110_0110
			// if combined.rotate().rotate().right().rotate().rotate() == s && combined.isPossible() && up.isPossible() && down.isPossible() {
			// 	count++
			// } else if combined.rotate().rotate().right().rotate().rotate() == s && !(combined.isPossible() && up.isPossible() && down.isPossible()) {
			// 	fmt.Println(s)
			// }
		}

		// findRecipeDepth(s, s, 0)
		// r := makeRecipe(s)
		// r.WriteTo(&buf)
		// if !r.check() {
		// 	count++
		// 	if count < 1000 {
		// 		fmt.Println("Error:", s, r.operation, r.original1, r.original2, "|", r.shape.recipe().original, findRecipeFromPreloaded(s).operation)
		// 	}

		// }
	}

	count = 0
	for s, r := range recipeMap {
		if !s.topLayer().hasCrystal() && r.Operation != "stack" {
			count++
			_ = r
		}
	}

	fmt.Println("Invalid recipes:", count)

	fHalfs, _ := os.Create("hardcoded-halfs.json")
	defer fHalfs.Close()
	json.NewEncoder(fHalfs).Encode(recipeMap)

	// fRecipes, _ := os.Create("recipes.bin")
	// defer fRecipes.Close()
	// fRecipes.Write(buf.Bytes())

	// fPins, _ := os.Create("hardcoded-pins.json")
	// defer fPins.Close()
	// json.NewEncoder(fPins).Encode(hardcodedPins)

	// fStacks, _ := os.Create("hardcoded-stacks.json")
	// defer fStacks.Close()
	// json.NewEncoder(fStacks).Encode(hardcodedStacks)

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

func findRecipe(s Shape) RecipeNode {
	// return findRecipeFromPreloaded(s)
	// if s.toFilled() <= 0b1111_1111_11111 {
	// 	return findRecipeFromPreloaded(s)
	// }

	if !s.topLayer().hasCrystal() {
		b, _ := s.unstack()
		return RecipeNode{
			Operation: "stack",
			Shape:     s,
			Original1: b,
		}
	}

	if s.isStackTopWithoutCrystals() {
		b, t := s.unstack()
		return RecipeNode{
			Operation: "stack",
			Shape:     s,
			Original1: b | t.crystals(),
		}
	}

	if s.isLeftRightValid() {
		right := (s & 0b0011_0011_0011_0011_0011_0011_0011_0011).collapse()
		left := (s &^ 0b0011_0011_0011_0011_0011_0011_0011_0011).collapse()
		if left.isPossible() && right.isPossible() {
			return RecipeNode{
				Operation: "combine",
				Shape:     s,
				Original1: s & 0b0011_0011_0011_0011_0011_0011_0011_0011,
				Original2: s &^ 0b0011_0011_0011_0011_0011_0011_0011_0011,
			}
		}
	}

	if p, ok := reversePinPush[s]; ok {
		return RecipeNode{
			Operation: "pin",
			Shape:     s,
			Original1: p,
		}
	}

	if r, ok := reverseRight[s]; ok {
		return RecipeNode{
			Operation: "right",
			Shape:     s,
			Original1: r,
		}
	}

	return RecipeNode{
		Operation: "rotate",
		Shape:     s,
		Original1: s.rotate().rotate().rotate(),
	}
}

func findRecipeDepth(original Shape, s Shape, depth int) int {
	if s == 0 {
		return 0
	}
	if s.layerCount() == 1 {
		return depth
	}

	if depth > 100 {
		fmt.Println("Circle:", original, s)
		return 0
	}

	r := findRecipeFromPreloaded(s)
	recipeMap[s] = r
	return max(findRecipeDepth(original, r.Original1, depth+1), findRecipeDepth(original, r.Original2, depth+1))
}

func checkRecipeMap() {
	for s := range recipeMap {
		stack := []Shape{s}

		for len(stack) > 0 {
			r := recipeMap[stack[len(stack)-1]]

			if r.Original1.toFilled()&^0b1111 != 0 {
				if r.Original1 == s {
					fmt.Println("Error recipe:", s)
					return
				}
				stack = append(stack, r.Original1)
			}
			if r.Original2.toFilled()&^0b1111 != 0 {
				if r.Original2 == s {
					fmt.Println("Error recipe:", s)
					return
				}
				stack = append(stack, r.Original2)
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
		rStack := []RecipeNode{r}

		for len(rStack) > 0 {
			if len(rStack) > 20000 {
				fmt.Println("Stack too big:", s)
				return
			}

			r := rStack[len(rStack)-1]
			rStack = rStack[:len(rStack)-1]

			if r.Original1.toFilled()&^0b1111 != 0 {
				newR, ok := recipeMap[r.Original1]
				if !ok {
					fmt.Println("No recipe:", s, r.Shape)
					return
				}
				rStack = append(rStack, newR)
			}

			if r.Original2.toFilled()&^0b1111 != 0 {
				newR, ok := recipeMap[r.Original2]
				if !ok {
					fmt.Println("No recipe:", s, r.Shape)
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

	fillRecipe(result.Original1)
	fillRecipe(result.Original2)
}

func findRecipeFromPreloaded(s Shape) RecipeNode {
	if s == 0 {
		return RecipeNode{Operation: "trivial", Shape: s}
	}
	if r, ok := recipeMap[s]; ok {
		return r
	}

	i, _ := slices.BinarySearchFunc(possibleRecipes, s, func(a recipe, b Shape) int {
		return int(a.shape) - int(b)
	})

	r := possibleRecipes[i]

	result := RecipeNode{Shape: s, Original1: r.original}

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
				result.Original2 = stackable
				break
			}
		}

		if op == "unknown" {
			if r.original&0b1100_1100_1100_1100_1100_1100_1100_1100 == 0 && (s & 0b1100_1100_1100_1100_1100_1100_1100_1100).isPossible() && (s&0b1100_1100_1100_1100_1100_1100_1100_1100) != 0 {
				op = "combine"
				result.Original2 = s & 0b1100_1100_1100_1100_1100_1100_1100_1100

			} else if r.original&^0b1100_1100_1100_1100_1100_1100_1100_1100 == 0 && (s &^ 0b1100_1100_1100_1100_1100_1100_1100_1100).isPossible() && (s&^0b1100_1100_1100_1100_1100_1100_1100_1100) != 0 {
				op = "combine"
				result.Original2 = s &^ 0b1100_1100_1100_1100_1100_1100_1100_1100
			}
		}
	}

	if op == "unknown" {
		fmt.Println("Unknown operation:", s)
		return RecipeNode{Operation: "unknown", Shape: s}
	}

	result.Operation = op
	return result
}

func printRecipe2(s Shape) {
	r := makeRecipe(s)
	for r.Operation != "unknown" && r.Operation != "trivial" && r.Operation != "half" {
		fmt.Println("Recipe:", r.Shape, "<-", r.Original1, r.Original2, r.Operation)
		r = makeRecipe(r.Original1)
	}
	fmt.Println("Recipe:", r.Shape, "<-", r.Original1, r.Original2, r.Operation)
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
