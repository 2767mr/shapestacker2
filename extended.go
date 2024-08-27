package main

import (
	"math/bits"
	"slices"
)

func (s Shape) mirror() Shape {
	mask1 := Shape(0x1111_1111)
	mask2 := Shape(0x8888_8888)
	mask3 := mask1 | mask2

	mask4 := Shape(0x2222_2222)
	mask5 := Shape(0x4444_4444)
	mask6 := mask4 | mask5

	s = s&^mask3 | (s&mask1)<<3 | (s&mask2)>>3
	s = s&^mask6 | (s&mask4)<<1 | (s&mask5)>>1
	return s
}

func (s Shape) minimal() Shape {
	return min(min(min(s, s.rotate()), min(s.rotate().rotate(), s.rotate().rotate().rotate())), min(min(s.mirror(), s.mirror().rotate()), min(s.mirror().rotate().rotate(), s.mirror().rotate().rotate().rotate())))
}

func (s Shape) isMinimal() bool {
	return s == s.minimal()
}

func (s Shape) topLayer() Shape {
	b := s | s>>16
	b |= (b & 0b0111_0111_0111_0111) << 1
	b |= (b & 0b0011_0011_0011_0011) << 2

	return (s >> ((3 - bits.LeadingZeros16(uint16(b))/4) * 4)) & 0b1111_0000_0000_0000_1111
}

func (s Shape) hasPins() bool {
	return (s&^(s<<16))&^0b1111_1111_1111_1111 != 0
}

func (s Shape) hasSpaces() bool {
	s = s.toFilled()
	return (s>>4)&^s != 0
}

func (s Shape) replaceCrystalsWithPins() Shape {
	return (s &^ (s >> 16)).collapse()
}

func (s Shape) isQuarter() bool {
	return s.minimal()&^0b0001_0001_0001_0001_0001_0001_0001_0001 == 0
}

func (s Shape) isPins() bool {
	return s>>16 != 0 && s&0b1111_1111_1111_1111 == 0
}

func (s Shape) isTrivialPinPusher() bool {
	b, t := s.unstackBottom()
	if !t.isPossible() {
		return false
	}
	l1, _ := t.unstackBottom()
	return b.isPins() && b.toFilled() == l1.toFilled()
}

func (s Shape) isLeftRightValid() bool {
	right := (s & 0b0011_0011_0011_0011_0011_0011_0011_0011).collapse()
	left := (s &^ 0b0011_0011_0011_0011_0011_0011_0011_0011).collapse()
	return (left|right) == s && left != 0 && right != 0
}

func (s Shape) isUpDownValid() bool {
	up := (s &^ 0b0110_0110_0110_0110_0110_0110_0110_0110).collapse()
	down := (s & 0b0110_0110_0110_0110_0110_0110_0110_0110).collapse()
	return (up|down) == s && up != 0 && down != 0
}

func (s Shape) crystals() Shape {
	s = s & (s >> 16)
	return s | (s << 16)
}

func (s Shape) isStackTopWithoutCrystals() bool {
	bottom, top := s.unstack()
	c := top.crystals()

	cBelow := c >> 4
	if s.toFilled()&cBelow.toFilled() != cBelow.toFilled() {
		return false
	}

	return top&^c != 0 && (bottom|c).stack(top&^c) == s
}

func (s Shape) recipe() recipe {
	i, ok := slices.BinarySearchFunc(possibleRecipes, s, func(a recipe, b Shape) int {
		return int(a.shape) - int(b)
	})
	if !ok {
		return recipe{}
	}

	return possibleRecipes[i]
}

func (s Shape) removeBottomEmpty() Shape {
	if s == 0 {
		return 0
	}
	bottom, top := s.unstackBottom()
	if bottom == 0 {
		return top.removeBottomEmpty()
	}
	return s
}
