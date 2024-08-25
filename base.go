package main

import "math/bits"

type shape uint32

func (s shape) String() string {
	var result string

	result += s.cornerAt(position_Layer0_Top_Right).String()
	result += s.cornerAt(position_Layer0_Bottom_Right).String()
	result += s.cornerAt(position_Layer0_Bottom_Left).String()
	result += s.cornerAt(position_Layer0_Top_Left).String()

	if s.layerCount() <= 1 {
		return result
	}

	result += ":"
	result += s.cornerAt(position_Layer1_Top_Right).String()
	result += s.cornerAt(position_Layer1_Bottom_Right).String()
	result += s.cornerAt(position_Layer1_Bottom_Left).String()
	result += s.cornerAt(position_Layer1_Top_Left).String()

	if s.layerCount() == 2 {
		return result
	}

	result += ":"
	result += s.cornerAt(position_Layer2_Top_Right).String()
	result += s.cornerAt(position_Layer2_Bottom_Right).String()
	result += s.cornerAt(position_Layer2_Bottom_Left).String()
	result += s.cornerAt(position_Layer2_Top_Left).String()

	if s.layerCount() == 3 {
		return result
	}

	result += ":"
	result += s.cornerAt(position_Layer3_Top_Right).String()
	result += s.cornerAt(position_Layer3_Bottom_Right).String()
	result += s.cornerAt(position_Layer3_Bottom_Left).String()
	result += s.cornerAt(position_Layer3_Top_Left).String()

	return result
}

func shapeFrom(s string) shape {
	var result shape

	index := position(0)
	odd := true
	for _, c := range s {
		if c == ':' {
			continue
		}

		odd = !odd

		if !odd {
			switch c {
			case 'C', 'R', 'W', 'S':
				result = result.setCornerAt(index, cornerTypeFilled)
			case 'P':
				result = result.setCornerAt(index, cornerTypePin)
			case 'c':
				result = result.setCornerAt(index, cornerTypeCrystal)
			}
			index++
		}
	}

	return result
}

type position byte

const (
	position_Layer0_Top_Left position = iota
	position_Layer0_Bottom_Left
	position_Layer0_Bottom_Right
	position_Layer0_Top_Right

	position_Layer1_Top_Left
	position_Layer1_Bottom_Left
	position_Layer1_Bottom_Right
	position_Layer1_Top_Right

	position_Layer2_Top_Left
	position_Layer2_Bottom_Left
	position_Layer2_Bottom_Right
	position_Layer2_Top_Right

	position_Layer3_Top_Left
	position_Layer3_Bottom_Left
	position_Layer3_Bottom_Right
	position_Layer3_Top_Right
)

func (p position) rotate() position {
	return (p+1)%4 + (p/4)*4
}

func (p position) layer() int {
	return int(p) / 4
}

func (p position) below() position {
	if p < 4 {
		return p
	}
	return p - 4
}

func (p position) above() position {
	if p >= 3*4 {
		return p
	}
	return p + 4
}

type cornerType byte

const (
	cornerTypeNone cornerType = iota
	cornerTypeFilled
	cornerTypePin
	cornerTypeCrystal
)

func (s shape) cornerAt(index position) cornerType {
	return cornerType(((s >> index) & 1) | ((s >> (index + 15)) & 0b10))
}

func (s shape) setCornerAt(index position, value cornerType) shape {
	return s&^(0b1<<index)&^(1<<(index+16)) | shape(value&1)<<index | shape(value&0b10)<<(index+15)
}

func (p cornerType) isEmpty() bool {
	return p == cornerTypeNone
}

func (p cornerType) isFilled() bool {
	return p == cornerTypeFilled
}

func (p cornerType) isPin() bool {
	return p == cornerTypePin
}

func (p cornerType) isCrystal() bool {
	return p == cornerTypeCrystal
}

func (p cornerType) String() string {
	switch p {
	case cornerTypeFilled:
		return "Cu"
	case cornerTypePin:
		return "P-"
	case cornerTypeCrystal:
		return "cu"
	}
	return "--"
}

// toFilled shape converts all pins and crystals into filled corners
func (s shape) toFilled() shape {
	return s&0b1111_1111_1111_1111 | (s >> 16)
}

func (s shape) layerCount() int {
	return 4 - bits.LeadingZeros16(uint16(s.toFilled()))/4
}

func (s shape) rotate() shape {
	return (s&0b1110_1110_1110_1110_1110_1110_1110_1110)>>1 | (s&0b0001_0001_0001_0001_0001_0001_0001_0001)<<3
}

func (s shape) hasCrystal() bool {
	return (s>>16)&s != 0
}

func (s shape) destoryCrystalAt(index position) shape {
	if !s.cornerAt(index).isCrystal() {
		return s
	}

	result := s.setCornerAt(index, cornerTypeNone)
	result = result.destoryCrystalAt(index.rotate())
	result = result.destoryCrystalAt(index.rotate().rotate().rotate())

	if index.layer() != 0 {
		result = result.destoryCrystalAt(index.below())
	}
	if index.layer() != 3 {
		result = result.destoryCrystalAt(index.above())
	}

	return result
}

func (s shape) collapse() shape {
	supported := s.supported()
	if supported == s {
		return s
	}

	withoutCrystals := s
	for i := position(0); i < 16; i++ {
		if supported.cornerAt(i).isEmpty() {
			withoutCrystals = withoutCrystals.destoryCrystalAt(i)
		}
	}

	mask := supported.toFilled()
	mask = mask | mask<<16

	fallen := withoutCrystals&mask | (withoutCrystals&^mask)>>4
	return fallen.collapse()
}

func (s shape) supported() shape {
	var supported shape
	newSupported := s & 0b1111_0000_0000_0000_1111

	for newSupported != supported {
		supported = newSupported
		for i := position(0); i < 16; i++ {
			if s.isSupported(i, supported) {
				newSupported = newSupported.setCornerAt(i, s.cornerAt(i))
			}
		}
	}

	return supported
}

func (s shape) unsupported() shape {
	unsupported := s &^ s.supported()
	for i := position(0); i < 16; i++ {
		if unsupported.cornerAt(i).isCrystal() {
			unsupported = unsupported.setCornerAt(i, cornerTypeNone)
		}
	}
	return unsupported
}

func (s shape) isSupported(position position, supported shape) bool {
	if s.cornerAt(position).isEmpty() {
		return false
	}

	if position.layer() == 0 {
		return true
	}

	if !supported.cornerAt(position).isEmpty() {
		return true
	}

	if !supported.cornerAt(position.below()).isEmpty() {
		return true
	}

	if !s.cornerAt(position).isPin() && !s.cornerAt(position.rotate()).isPin() && !supported.cornerAt(position.rotate()).isEmpty() {
		return true
	}

	if !s.cornerAt(position).isPin() && !s.cornerAt(position.rotate().rotate().rotate()).isPin() && !supported.cornerAt(position.rotate().rotate().rotate()).isEmpty() {
		return true
	}

	if s.cornerAt(position).isCrystal() && supported.cornerAt(position.above()).isCrystal() {
		return true
	}

	return false
}

func (s shape) pushPins() shape {
	for i := position(12); i < 16; i++ {
		s = s.destoryCrystalAt(i)
	}

	pins := s.toFilled() & 0b1111
	s <<= 4
	s &^= 0b1111_0000_0000_0000
	s |= pins << 16

	return s.collapse()
}

func (s shape) right() shape {
	for i := position(0); i < 4; i++ {
		if s.cornerAt(position_Layer0_Top_Right+i*4).isCrystal() && s.cornerAt(position_Layer0_Top_Left+i*4).isCrystal() {
			s = s.destoryCrystalAt(position_Layer0_Top_Right + i*4)
		}

		if s.cornerAt(position_Layer0_Bottom_Right+i*4).isCrystal() && s.cornerAt(position_Layer0_Bottom_Left+i*4).isCrystal() {
			s = s.destoryCrystalAt(position_Layer0_Bottom_Right + i*4)
		}
	}

	right := s & 0b1100_1100_1100_1100_1100_1100_1100_1100
	return right.collapse()
}

func (s shape) combine(with shape) shape {
	return s | with
}

//stack other shape on top of s
func (s shape) stack(other shape) shape {
	for i := position(0); i < 16; i++ {
		other = other.destoryCrystalAt(i)
	}

	for other != 0 {
		group := other.firstGroup()
		other = other &^ group

		for group.toFilled()&0b1111 == 0 && group != 0 {
			group >>= 4
		}

		filledGroup := group.toFilled()
		filledS := s.toFilled()

		valid := 0
		for i := 3; i >= 0; i-- {
			if (filledGroup<<(i*4))&filledS != 0 {
				valid = i + 1
				break
			}
		}

		s = s | ((group << (valid * 4)) & 0b1111_1111_1111_1111)
		s = s | ((group &^ 0b1111_1111_1111_1111) << (valid * 4))
	}

	return s
}

func (s shape) firstGroup() shape {
	for i := position(0); i < 16; i++ {
		if !s.cornerAt(i).isEmpty() {
			group := s.connectedGroup(i, 0)

			groupFilled := group.toFilled()
			leftOver := s.toFilled() &^ groupFilled

			if groupFilled>>4&leftOver == 0 && groupFilled>>8&leftOver == 0 && groupFilled>>12&leftOver == 0 {
				return group
			}
		}
	}
	return 0
}

func (s shape) connectedGroup(position position, group shape) shape {
	if s.cornerAt(position).isEmpty() {
		return group
	}

	if group.cornerAt(position).isFilled() {
		return group
	}

	if s.cornerAt(position).isPin() {
		return group.setCornerAt(position, cornerTypePin)
	}

	group = group.setCornerAt(position, s.cornerAt(position))

	if s.cornerAt(position.rotate()).isFilled() {
		group = s.connectedGroup(position.rotate(), group)
	}
	if s.cornerAt(position.rotate().rotate().rotate()).isFilled() {
		group = s.connectedGroup(position.rotate().rotate().rotate(), group)
	}
	if s.cornerAt(position.above()).isFilled() {
		group = s.connectedGroup(position.above(), group)
	}
	if s.cornerAt(position).isCrystal() && s.cornerAt(position.below()).isCrystal() {
		group = s.connectedGroup(position.below(), group)
	}
	return group
}

func (s shape) crystalGenerator() shape {
	layers := s.layerCount()
	for i := position(0); i < position(layers*4); i++ {
		if s.cornerAt(i).isEmpty() || s.cornerAt(i).isPin() {
			s = s.setCornerAt(i, cornerTypeCrystal)
		}
	}
	return s
}
