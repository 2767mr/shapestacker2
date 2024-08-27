package main

import (
	"testing"
)

type TestCase struct {
	name      string
	input     Shape
	input2    Shape
	expect    Shape
	operation func(*testing.T, TestCase)
}

var (
	checkPins = func(t *testing.T, tc TestCase) {
		result := tc.input.pushPins()
		pass := result == tc.expect
		if !pass {
			t.Fatal(" Expected", tc.expect, "but got", result, "original", tc.input)
		}
	}

	checkCrystal = func(t *testing.T, tc TestCase) {
		result := tc.input.crystalGenerator()
		pass := result == tc.expect
		if !pass {
			t.Fatal(" Expected", tc.expect, "but got", result, "original", tc.input)
		}
	}

	checkCutLeft = func(t *testing.T, tc TestCase) {
		result := tc.input.right()
		pass := result == tc.expect
		if !pass {
			t.Fatal(" Expected", tc.expect, "but got", result, "original", tc.input)
		}
	}

	checkCutRight = func(t *testing.T, tc TestCase) {
		result := tc.input.rotate().rotate().right().rotate().rotate()
		pass := result == tc.expect
		if !pass {
			t.Fatal(" Expected", tc.expect, "but got", result, "original", tc.input)
		}
	}

	checkStack = func(t *testing.T, tc TestCase) {
		result := tc.input2.stack(tc.input)
		pass := result == tc.expect
		if !pass {
			t.Fatal(" Expected", tc.expect, "but got", result, "original", tc.input2, "+", tc.input)
		}
	}
)

func testOne(t *testing.T, tc TestCase) {
	t.Run(tc.name, func(t *testing.T) {
		tc.operation(t, tc)
	})
}

func TestShapes(t *testing.T) {
	testOne(t, TestCase{"PP_01", 0x00000001, 0, 0x00010010, checkPins})
	testOne(t, TestCase{"PP_02", 0x00030030, 0, 0x00330300, checkPins})
	testOne(t, TestCase{"PP_03", 0x0000f931, 0, 0x00019310, checkPins})
	testOne(t, TestCase{"PP_04", 0x11701571, 0, 0x00010014, checkPins})
	testOne(t, TestCase{"PP_05", 0xcacfffff, 0, 0x000f0170, checkPins})
	testOne(t, TestCase{"PP_06", 0x22222273, 0, 0x00030114, checkPins})

	testOne(t, TestCase{"PP_08", shapeFrom("--P-Cu--:--P-cuCu:Cucu----:--cu----"), 0, shapeFrom("CuP-P---:--P-Cu--:--P-cuCu"), checkPins})

	testOne(t, TestCase{"CRYSTAL_01", 0x00000001, 0, 0x000e000f, checkCrystal})
	testOne(t, TestCase{"CRYSTAL_02", 0x00010010, 0, 0x00ef00ff, checkCrystal})

	testOne(t, TestCase{"CUT_01", 0x936c, 0, 0x00cc, checkCutLeft})
	testOne(t, TestCase{"CUT_02", 0x936c, 0, 0x0132, checkCutRight})
	testOne(t, TestCase{"CUT_03", 0x000f0000, 0, 0x000c0000, checkCutLeft})
	testOne(t, TestCase{"CUT_04", 0x000f0000, 0, 0x00030000, checkCutRight})
	testOne(t, TestCase{"CUT_05", 0x000f000f, 0, 0x0000, checkCutLeft})
	testOne(t, TestCase{"CUT_06", 0x000f000f, 0, 0x0000, checkCutRight})
	testOne(t, TestCase{"CUT_07", 0xe8c4f8c4, 0, 0x0000, checkCutLeft})
	testOne(t, TestCase{"CUT_08", 0xe8c4f8c4, 0, 0x0001, checkCutRight})
	testOne(t, TestCase{"CUT_09", 0x00500073, 0, 0x0000, checkCutLeft})
	testOne(t, TestCase{"CUT_10", 0x00500073, 0, 0x00100033, checkCutRight})
	testOne(t, TestCase{"CUT_11", 0x005e00ff, 0, 0x0008, checkCutLeft})
	testOne(t, TestCase{"CUT_12", 0x005e00ff, 0, 0x00100031, checkCutRight})

	testOne(t, TestCase{"STACK_01", 0xafff, 0x1115, 0x1115, checkStack})
	testOne(t, TestCase{"STACK_02", 0x00010000, 0x1111, 0x1111, checkStack})
	testOne(t, TestCase{"STACK_03", 0x000f, 0x00010000, 0x000100f0, checkStack})
	testOne(t, TestCase{"STACK_04", 0x00100001, 0x00010110, 0x00011110, checkStack})
	testOne(t, TestCase{"STACK_05", 0x000f0000, 0x08ce, 0x842108ce, checkStack})
	testOne(t, TestCase{"STACK_06", 0x000f005f, 0x000a, 0x0000000f, checkStack})
	testOne(t, TestCase{"STACK_07", 0xfffa, 0x1115, 0x111f, checkStack})
}
