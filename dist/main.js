// main.ts
var hardcodedPins = load("./hardcoded-pins.json");
var hardcodedStacks = load("./hardcoded-stacks.json");
var hardcodedRecipes = load("./hardcoded-halfs.json").then(
  (recipes) => Object.fromEntries(Object.entries(recipes).map(([shape, recipe]) => [shape, { shape: +shape, ...recipe }]))
);
async function main() {
  globalThis.solve = async (shape) => {
    const makeRecipe2 = solver(await hardcodedPins, await hardcodedStacks, await hardcodedRecipes);
    function printRecipe2(shape2, level, mirrored = false) {
      let result = "";
      const recipe = makeRecipe2(shape2);
      if (recipe.operation == "mirror") {
        return printRecipe2(mirror(shape2), level, !mirrored);
      }
      if (mirrored && recipe.operation == "right") {
        recipe.operation = "left";
      }
      function printShape(s) {
        return toString(mirrored ? mirror(s) : s);
      }
      if (recipe.original1 && recipe.original2) {
        result += "&nbsp;&nbsp;&nbsp;&nbsp;".repeat(level) + printShape(shape2) + ' = <span style="color:blue">' + recipe.operation + '</span> <span style="color:magenta">' + printShape(recipe.original1) + '</span> + </span> <span style="color:red">' + printShape(recipe.original2) + "</span><br />";
        if (toFilled(recipe.original1) > 15 && toFilled(recipe.original2) > 15) {
          level++;
        }
        if (toFilled(recipe.original1) > 15) {
          result += printRecipe2(recipe.original1, level);
        }
        if (toFilled(recipe.original2) > 15) {
          result += printRecipe2(recipe.original2, level);
        }
      } else if (recipe.original1) {
        result += "&nbsp;&nbsp;&nbsp;&nbsp;".repeat(level) + printShape(shape2) + ' = <span style="color:blue">' + recipe.operation + '</span> <span style="color:magenta">' + printShape(recipe.original1) + "</span><br />";
        if (toFilled(recipe.original1) > 15) {
          result += printRecipe2(recipe.original1, level);
        }
      } else {
        result += "&nbsp;&nbsp;&nbsp;&nbsp;".repeat(level) + printShape(shape2) + ' = <span style="color:blue">' + recipe.operation + "</span><br />";
      }
      return result;
    }
    const output = document.getElementById("output");
    output.innerHTML = printRecipe2(parseShape(shape), 0);
  };
  const makeRecipe = solver(await hardcodedPins, await hardcodedStacks, await hardcodedRecipes);
  function printRecipe(shape, level) {
    let result = "";
    const recipe = makeRecipe(shape);
    if (recipe.original1 && recipe.original2) {
      result += "  ".repeat(level) + toString(shape) + " = " + recipe.operation + " " + toString(recipe.original1) + " + " + toString(recipe.original2) + "\n";
    } else if (recipe.original1) {
      result += "  ".repeat(level) + toString(shape) + " = " + recipe.operation + " " + toString(recipe.original1) + "\n";
    } else {
      result += "  ".repeat(level) + toString(shape) + " = " + recipe.operation + "\n";
    }
    if (recipe.original1 && toFilled(recipe.original1) > 15) {
      result += printRecipe(recipe.original1, level);
      level++;
    }
    if (recipe.original2 && toFilled(recipe.original2) > 15) {
      result += printRecipe(recipe.original2, level);
    }
    return result;
  }
  console.log(printRecipe(mirror(parseShape("----CuCu:----Cu--:----P-cr:----Cucr")), 0));
}
main();
console.log(viewer);
async function load(file) {
  if ("process" in globalThis) {
    const fs = globalThis["require"]("node:fs");
    return new Promise((resolve, reject) => {
      fs.readFile(file, (err, data) => {
        if (err) {
          reject(err);
        } else {
          resolve(JSON.parse(data));
        }
      });
    });
  }
  return fetch(file).then((r) => r.json());
}
function parseShape(text) {
  let result = 0;
  let index = 0;
  let odd = true;
  for (let i = 0; i < text.length; i++) {
    const c = text[i];
    if (c == ":") {
      continue;
    }
    odd = !odd;
    if (!odd) {
      switch (c) {
        case "C":
        case "R":
        case "W":
        case "S":
          result = setCornerAt(result, 3 - index % 4 + (index / 4 >>> 0) * 4, 1);
          break;
        case "P":
          result = setCornerAt(result, 3 - index % 4 + (index / 4 >>> 0) * 4, 2);
          break;
        case "c":
          result = setCornerAt(result, 3 - index % 4 + (index / 4 >>> 0) * 4, 3);
          break;
      }
      index++;
    }
  }
  return result >>> 0;
}
function setCornerAt(shape, index, value) {
  return shape & ~(1 << index >>> 0) & ~(1 << index + 16) | (value & 1) << index >>> 0 | (value & 2) << index + 15 >>> 0;
}
function cornerAt(s, index) {
  return s >>> index & 1 | s >>> index + 15 & 2;
}
function toString(s) {
  const filled = (s | s >>> 16) & 65535;
  let result = "";
  result += cornerString(s, 3);
  result += cornerString(s, 2);
  result += cornerString(s, 1);
  result += cornerString(s, 0);
  if (filled <= 15) {
    return result;
  }
  result += ":";
  result += cornerString(s, 7);
  result += cornerString(s, 6);
  result += cornerString(s, 5);
  result += cornerString(s, 4);
  if (filled <= 255) {
    return result;
  }
  result += ":";
  result += cornerString(s, 11);
  result += cornerString(s, 10);
  result += cornerString(s, 9);
  result += cornerString(s, 8);
  if (filled <= 4095) {
    return result;
  }
  result += ":";
  result += cornerString(s, 15);
  result += cornerString(s, 14);
  result += cornerString(s, 13);
  result += cornerString(s, 12);
  return result;
}
function cornerString(s, corner) {
  switch (cornerAt(s, corner)) {
    case 1:
      return "Cu";
    case 2:
      return "P-";
    case 3:
      return "cu";
    default:
      return "--";
  }
}
function toFilled(s) {
  return (s | s >>> 16) & 65535;
}
function mirror(s) {
  const mask1 = 286331153;
  const mask2 = 2290649224;
  const mask3 = mask1 | mask2;
  const mask4 = 572662306;
  const mask5 = 1145324612;
  const mask6 = mask4 | mask5;
  s = s & ~mask3 | (s & mask1) << 3 | (s & mask2) >>> 3;
  s = s & ~mask6 | (s & mask4) << 1 | (s & mask5) >>> 1;
  return s;
}
function solver(hardcodedPins2, hardcodedStacks2, hardcodedRecipes2) {
  return makeRecipe;
  function topLayer(s) {
    let b = toFilled(s);
    b |= (b & 30583) << 1;
    b |= (b & 13107) << 2;
    return s >>> (3 - (Math.clz32(b) - 16) / 4 >>> 0) * 4 & 983055;
  }
  function hasCrystal(s) {
    return (s >>> 16 & s) !== 0;
  }
  function layerCount(s) {
    return 4 - ((Math.clz32(toFilled(s)) - 16) / 4 >>> 0);
  }
  function unstack(s) {
    if (s == 0) {
      return [0, 0];
    }
    const mask = 983055 << (layerCount(s) - 1) * 4 >>> 0;
    return [s & ~mask, s & mask];
  }
  function isLeftRightValid(s) {
    const right = collapse(s & 858993459);
    const left = collapse(s & ~858993459);
    return (left | right) >>> 0 == s >>> 0 && left != 0 && right != 0;
  }
  function isUpDownValid(s) {
    const up = collapse(s & ~1717986918);
    const down = collapse(s & 1717986918);
    return (down | up) >>> 0 == s >>> 0 && down != 0 && up != 0;
  }
  function supported(s) {
    let sup = 0;
    let newSupported = s & 983055;
    while (newSupported != sup) {
      sup = newSupported;
      for (let i = 0; i < 16; i++) {
        if (isSupported(s, i, sup)) {
          newSupported = setCornerAt(newSupported, i, cornerAt(s, i));
        }
      }
    }
    return sup;
  }
  function isSupported(s, position, sup) {
    if (cornerAt(s, position) === 0) {
      return false;
    }
    if (position / 4 >>> 0 == 0) {
      return true;
    }
    if (cornerAt(sup, position) !== 0) {
      return true;
    }
    if (cornerAt(sup, below(position)) != 0) {
      return true;
    }
    if (cornerAt(s, position) != 2 && cornerAt(s, rotatePosition(position)) != 2 && cornerAt(sup, rotatePosition(position)) != 0) {
      return true;
    }
    if (cornerAt(s, position) != 2 && cornerAt(s, rotatePosition(rotatePosition(rotatePosition(position)))) != 2 && cornerAt(sup, rotatePosition(rotatePosition(rotatePosition(position)))) != 0) {
      return true;
    }
    if (cornerAt(s, position) == 3 && cornerAt(sup, above(position)) == 3) {
      return true;
    }
    return false;
  }
  function rotatePosition(p) {
    return (p + 1) % 4 + (p / 4 >>> 0) * 4;
  }
  function below(p) {
    if (p < 4) {
      return p;
    }
    return p - 4;
  }
  function above(p) {
    if (p >= 3 * 4) {
      return p;
    }
    return p + 4;
  }
  function firstGroup(s) {
    for (let i = 0; i < 16; i++) {
      if (cornerAt(s, i) != 0) {
        const group = connectedGroup(s, i, 0);
        const groupFilled = toFilled(group);
        const leftOver = toFilled(s) & ~groupFilled;
        if ((groupFilled >>> 4 & leftOver) == 0 && (groupFilled >>> 8 & leftOver) == 0 && (groupFilled >>> 12 & leftOver) == 0) {
          return group;
        }
      }
    }
    return 0;
  }
  function connectedGroup(s, position, group) {
    if (cornerAt(s, position) == 0) {
      return group;
    }
    if (cornerAt(group, position) == 1 || cornerAt(group, position) == 3) {
      return group;
    }
    if (cornerAt(s, position) == 2) {
      return setCornerAt(group, position, 2);
    }
    group = setCornerAt(group, position, cornerAt(s, position));
    if (cornerAt(s, rotatePosition(position)) == 1) {
      group = connectedGroup(s, rotatePosition(position), group);
    }
    if (cornerAt(s, rotatePosition(rotatePosition(rotatePosition(position)))) == 1) {
      group = connectedGroup(s, rotatePosition(rotatePosition(rotatePosition(position))), group);
    }
    return group;
  }
  function collapse(s) {
    const sup = supported(s);
    if (sup == s) {
      return s;
    }
    let unsupported = s & ~sup;
    const crystals2 = unsupported & unsupported >>> 16;
    unsupported &= ~(crystals2 | crystals2 << 16) >>> 0;
    let result = sup;
    while (unsupported != 0) {
      const group = firstGroup(unsupported);
      unsupported = unsupported & ~group;
      let valid = 3;
      for (let i = 0; i < 4; i++) {
        if ((toFilled(group) >>> i * 4 & toFilled(result)) != 0 || group >>> i * 4 << i * 4 >>> 0 != group) {
          valid = i - 1;
          break;
        }
      }
      result = result | (group & 65535) >>> valid * 4;
      result = result | group >>> valid * 4 & ~65535;
    }
    return result;
  }
  function destoryCrystalAt(s, index) {
    if (cornerAt(s, index) != 3) {
      return s;
    }
    let result = setCornerAt(s, index, 0);
    result = destoryCrystalAt(result, rotatePosition(index));
    result = destoryCrystalAt(result, rotatePosition(rotatePosition(rotatePosition(index))));
    if (index / 4 >>> 0 != 0) {
      result = destoryCrystalAt(result, below(index));
    }
    if (index / 4 >>> 0 != 3) {
      result = destoryCrystalAt(result, above(index));
    }
    return result;
  }
  function stack(bottom, top) {
    for (let i = 0; i < 16; i++) {
      top = destoryCrystalAt(top, i);
    }
    let mask = 0;
    while (top != 0) {
      let group = firstGroup(top);
      top = top & ~group;
      while ((toFilled(group) & 15) == 0 && group != 0) {
        group >>>= 4;
      }
      const filledGroup = toFilled(group);
      if ((filledGroup & mask) != 0 || (filledGroup << 12 & toFilled(bottom)) != 0) {
        mask |= toFilled(group);
        continue;
      }
      const filledS = toFilled(bottom);
      let valid = 0;
      for (let i = 3; i >= 0; i--) {
        if ((filledGroup << i * 4 & filledS) != 0) {
          valid = i + 1;
          break;
        }
      }
      bottom = bottom | group << valid * 4 & 65535;
      bottom = bottom | (group & ~65535) << valid * 4;
    }
    return bottom;
  }
  function rotate(s) {
    return (s & 4008636142) >>> 1 | (s & 286331153) << 3;
  }
  function minimal(s) {
    return Math.min(
      s >>> 0,
      rotate(s) >>> 0,
      rotate(rotate(s)) >>> 0,
      rotate(rotate(rotate(s))) >>> 0,
      mirror(s) >>> 0,
      mirror(rotate(s)) >>> 0,
      mirror(rotate(rotate(s))) >>> 0,
      mirror(rotate(rotate(rotate(s)))) >>> 0
    );
  }
  function isMinimal(s) {
    return s == minimal(s);
  }
  function crystals(s) {
    s = s & s >>> 16;
    return s | s << 16;
  }
  function unstackBottom(s) {
    const mask = 983055;
    return [s & mask, (s & ~mask) >>> 4];
  }
  function removeBottomEmpty(s) {
    if (s == 0) {
      return 0;
    }
    const [bottom, top] = unstackBottom(s);
    if (bottom == 0) {
      return removeBottomEmpty(top);
    }
    return s;
  }
  function isStackTopWithoutCrystals(s) {
    const [bottom, top] = unstack(s);
    const c = crystals(top);
    const cBelow = c >>> 4;
    if ((toFilled(s) & toFilled(cBelow)) != toFilled(cBelow)) {
      return false;
    }
    return (top & ~c) != 0 && stack(bottom | c, top & ~c) >>> 0 == s >>> 0;
  }
  function makeRecipe(s) {
    if (hardcodedRecipes2[s]) {
      return hardcodedRecipes2[s];
    }
    if (toFilled(s) <= 15) {
      return {
        shape: s,
        operation: "trivial"
      };
    }
    if (!hasCrystal(topLayer(s))) {
      const [bottom, top] = unstack(s);
      return {
        operation: "stack",
        shape: s,
        original1: bottom,
        original2: top >>> (layerCount(s) - 1) * 4
      };
    }
    if ((s & ~858993459) == 0) {
      return { shape: s, operation: "half" };
    }
    if (isLeftRightValid(s)) {
      const right = s & 858993459;
      const left = s & ~858993459;
      if (collapse(left) == left && collapse(right) == right) {
        return { shape: s, operation: "combine", original1: left, original2: right };
      }
    }
    if (isUpDownValid(s)) {
      return { shape: s, operation: "rotate", original1: rotate(rotate(rotate(s))) };
    }
    if (!isMinimal(s)) {
      if (isMinimal(rotate(s)) || isMinimal(rotate(rotate(s))) || isMinimal(rotate(rotate(rotate(s))))) {
        return { shape: s, operation: "rotate", original1: rotate(rotate(rotate(s))) };
      } else {
        return { shape: s, operation: "mirror", original1: mirror(s) };
      }
    }
    if (isStackTopWithoutCrystals(s)) {
      let [bottom, top] = unstack(s);
      bottom |= crystals(top);
      return {
        operation: "stack",
        shape: s,
        original1: bottom,
        original2: removeBottomEmpty(top & ~bottom)
      };
    }
    const pin = hardcodedPins2[s];
    if (pin) {
      return { shape: s, operation: "pushPins", original1: pin };
    }
    const stack2 = hardcodedStacks2[s];
    if (stack2) {
      return { shape: s, operation: "stack", original1: stack2, original2: removeBottomEmpty(s & ~stack2) };
    }
    return { shape: s, operation: "unknown" };
  }
}
