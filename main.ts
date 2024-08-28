export { };
const hardcodedPins = load('./hardcoded-pins.json');
const hardcodedStacks = load('./hardcoded-stacks.json');
const hardcodedRecipes = load('./hardcoded-halfs.json')
    .then((recipes: Record<string, Omit<Recipe, 'shape'>>) =>
        Object.fromEntries(Object.entries(recipes)
            .map(([shape, recipe]) => [shape, { shape: +shape, ...recipe } as Recipe]))
    );

async function main() {
    globalThis.solve = async (shape: string) => {
        const makeRecipe = solver(await hardcodedPins, await hardcodedStacks, await hardcodedRecipes)
        function printRecipe(shape: Shape, level: number, mirrored = false): string {
            let result = '';
            const recipe = makeRecipe(shape);
            if (recipe.operation == 'mirror') {
                return printRecipe(mirror(shape), level, !mirrored)
            }

            function printShape(s: Shape) {
                return toString(mirrored ? mirror(s) : s)
            }

            if (recipe.original1 && recipe.original2) {
                result += ('&nbsp;&nbsp;&nbsp;&nbsp;'.repeat(level) + printShape(shape) + ' = <span style="color:blue">' + recipe.operation + '</span> <span style="color:magenta">' + printShape(recipe.original1) + '</span> + </span> <span style="color:red">' + printShape(recipe.original2)) + '</span><br />';

                if (toFilled(recipe.original1) > 0b1111 && toFilled(recipe.original2) > 0b1111) {
                    level++
                }

                if (toFilled(recipe.original1) > 0b1111) {
                    result += printRecipe(recipe.original1, level)
                }
                if (toFilled(recipe.original2) > 0b1111) {
                    result += printRecipe(recipe.original2, level)
                }
            } else if (recipe.original1) {
                result += ('&nbsp;&nbsp;&nbsp;&nbsp;'.repeat(level) + printShape(shape) + ' = <span style="color:blue">' + recipe.operation + '</span> <span style="color:magenta">' + printShape(recipe.original1)) + '</span><br />';
                if (toFilled(recipe.original1) > 0b1111) {
                    result += printRecipe(recipe.original1, level)
                }
            } else {
                result += ('&nbsp;&nbsp;&nbsp;&nbsp;'.repeat(level) + printShape(shape) + ' = <span style="color:blue">' + recipe.operation) + '</span><br />';
            }

            return result
        }



        const output = document.getElementById('output') as HTMLDivElement;
        output.innerHTML = printRecipe(parseShape(shape), 0)
    };
    const makeRecipe = solver(await hardcodedPins, await hardcodedStacks, await hardcodedRecipes)
    function printRecipe(shape: Shape, level: number): string {
        let result = '';
        const recipe = makeRecipe(shape);
        if (recipe.original1 && recipe.original2) {
            result += ('  '.repeat(level) + toString(shape) + ' = ' + recipe.operation + ' ' + toString(recipe.original1) + ' + ' + toString(recipe.original2)) + '\n';
        } else if (recipe.original1) {
            result += ('  '.repeat(level) + toString(shape) + ' = ' + recipe.operation + ' ' + toString(recipe.original1)) + '\n';
        } else {
            result += ('  '.repeat(level) + toString(shape) + ' = ' + recipe.operation) + '\n';
        }

        if (recipe.original1 && toFilled(recipe.original1) > 0b1111) {
            result += printRecipe(recipe.original1, level)
            level++
        }
        if (recipe.original2 && toFilled(recipe.original2) > 0b1111) {
            result += printRecipe(recipe.original2, level)
        }

        return result
    }


    // printRecipe(parseShape('----CuCu:----Cu--:----P-cr:----Cucr'), 0)
    console.log(printRecipe(mirror(parseShape('----CuCu:----Cu--:----P-cr:----Cucr')), 0))
}
main()

console.log(viewer);

async function load(file: string): Promise<any> {
    if ('process' in globalThis) {
        const fs = globalThis['require']('node:fs');
        return new Promise((resolve, reject) => {
            fs.readFile(file, (err, data) => {
                if (err) {
                    reject(err)
                } else {
                    resolve(JSON.parse(data as unknown as string))
                }
            })
        })
    }

    return fetch(file).then(r => r.json())
}

type Shape = number

interface Recipe {
    shape: Shape
    operation: string
    original1?: Shape
    original2?: Shape
}

function parseShape(text: string): Shape {
    let result: Shape = 0

    let index = 0;
    let odd = true
    for (let i = 0; i < text.length; i++) {
        const c = text[i];
        if (c == ':') {
            continue;
        }

        odd = !odd;

        if (!odd) {
            switch (c) {
                case 'C':
                case 'R':
                case 'W':
                case 'S':
                    result = setCornerAt(result, (3 - index % 4) + ((index / 4) >>> 0) * 4, 0b01)
                    break;
                case 'P':
                    result = setCornerAt(result, (3 - index % 4) + ((index / 4) >>> 0) * 4, 0b10)
                    break;
                case 'c':
                    result = setCornerAt(result, (3 - index % 4) + ((index / 4) >>> 0) * 4, 0b11)
                    break;
            }
            index++
        }
    }
    return result >>> 0
}

function setCornerAt(shape: Shape, index: number, value: number): Shape {
    return shape & ~((0b1 << index) >>> 0) & ~(1 << (index + 16)) | ((value & 1) << index) >>> 0 | ((value & 0b10) << (index + 15)) >>> 0
}


function cornerAt(s: Shape, index: number): number {
    return ((s >>> index) & 1) | ((s >>> (index + 15)) & 0b10)
}

function toString(s: Shape) {
    const filled = ((s | s >>> 16) & 0b1111_1111_1111_1111);

    let result = ''

    result += cornerString(s, 3)
    result += cornerString(s, 2)
    result += cornerString(s, 1)
    result += cornerString(s, 0)

    if (filled <= 0b1111) {
        return result
    }

    result += ':'
    result += cornerString(s, 7)
    result += cornerString(s, 6)
    result += cornerString(s, 5)
    result += cornerString(s, 4)

    if (filled <= 0b1111_1111) {
        return result
    }

    result += ':'
    result += cornerString(s, 11)
    result += cornerString(s, 10)
    result += cornerString(s, 9)
    result += cornerString(s, 8)

    if (filled <= 0b1111_1111_1111) {
        return result
    }

    result += ':'
    result += cornerString(s, 15)
    result += cornerString(s, 14)
    result += cornerString(s, 13)
    result += cornerString(s, 12)

    return result
}

function cornerString(s: Shape, corner: number): string {
    switch (cornerAt(s, corner)) {
        case 0b01:
            return 'Cu';
        case 0b10:
            return 'P-';
        case 0b11:
            return 'cu';
        default:
            return '--';
    }
}

async function readSection(file: string, start: number, end: number): Promise<Uint8Array> {
    const fs = globalThis['require']('node:fs');
    return new Promise<Uint8Array>((resolve, reject) => {
        const buffer = new Uint8Array(end - start)
        const fileStream = fs.createReadStream(file, { start, end: end - 1 });
        let offset = 0;
        fileStream.on('data', (chunk: Buffer) => {
            buffer.set(chunk, offset);
            offset += chunk.length;
        });
        fileStream.on('end', () => {
            if (offset != buffer.length) {
                const newBuffer = new Uint8Array(offset);
                newBuffer.set(buffer.slice(0, offset));
                resolve(newBuffer)
            } else {
                resolve(buffer)
            }
        });
        fileStream.on('error', (err) => {
            reject(err)
        });
    });
}

function toFilled(s: Shape): number {
    return (s | (s >>> 16)) & 0b1111_1111_1111_1111;
}

function mirror(s: Shape): Shape {
    const mask1 = 0x1111_1111
    const mask2 = 0x8888_8888
    const mask3 = mask1 | mask2

    const mask4 = 0x2222_2222
    const mask5 = 0x4444_4444
    const mask6 = mask4 | mask5

    s = (s & ~mask3) | ((s & mask1) << 3) | ((s & mask2) >>> 3)
    s = (s & ~mask6) | ((s & mask4) << 1) | ((s & mask5) >>> 1)
    return s
}



function solver(hardcodedPins: Record<Shape, Shape>, hardcodedStacks: Record<Shape, Shape>, hardcodedRecipes: Record<Shape, Recipe>): (s: Shape) => Recipe {
    return makeRecipe;

    function topLayer(s: Shape): Shape {
        let b = toFilled(s);
        b |= (b & 0b0111_0111_0111_0111) << 1;
        b |= (b & 0b0011_0011_0011_0011) << 2;

        return (s >>> ((3 - ((Math.clz32(b) - 16) / 4) >>> 0) * 4)) & 0b1111_0000_0000_0000_1111
    }

    function hasCrystal(s: Shape): boolean {
        return ((s >>> 16) & s) !== 0
    }
    function layerCount(s: Shape): number {
        return 4 - (((Math.clz32(toFilled(s)) - 16) / 4) >>> 0)
    }
    function unstack(s: Shape): [bottom: Shape, top: Shape] {
        if (s == 0) {
            return [0, 0]
        }
        const mask = (0b1111_0000_0000_0000_1111 << ((layerCount(s) - 1) * 4)) >>> 0
        return [s & ~mask, s & mask]
    }

    function isLeftRightValid(s: Shape): boolean {
        const right = collapse(s & 0b0011_0011_0011_0011_0011_0011_0011_0011)
        const left = collapse(s & ~0b0011_0011_0011_0011_0011_0011_0011_0011)
        return ((left | right) >>> 0) == (s >>> 0) && left != 0 && right != 0
    }

    function isUpDownValid(s: Shape): boolean {
        const up = collapse(s & ~0b0110_0110_0110_0110_0110_0110_0110_0110)
        const down = collapse(s & 0b0110_0110_0110_0110_0110_0110_0110_0110)
        return ((down | up) >>> 0) == (s >>> 0) && down != 0 && up != 0
    }

    function supported(s: Shape): Shape {
        let sup: Shape = 0
        let newSupported = s & 0b1111_0000_0000_0000_1111

        while (newSupported != sup) {
            sup = newSupported
            for (let i = 0; i < 16; i++) {
                if (isSupported(s, i, sup)) {
                    newSupported = setCornerAt(newSupported, i, cornerAt(s, i))
                }
            }
        }

        return sup
    }

    function isSupported(s: Shape, position: number, sup: Shape): boolean {
        if (cornerAt(s, position) === 0b00) {
            return false
        }

        if (((position / 4) >>> 0) == 0) {
            return true
        }

        if (cornerAt(sup, position) !== 0b00) {
            return true
        }

        if (cornerAt(sup, below(position)) != 0b00) {
            return true
        }

        if (cornerAt(s, position) != 0b10 && cornerAt(s, rotatePosition(position)) != 0b10 && cornerAt(sup, rotatePosition(position)) != 0b00) {
            return true
        }

        if (cornerAt(s, position) != 0b10 && cornerAt(s, rotatePosition(rotatePosition(rotatePosition(position)))) != 0b10 && cornerAt(sup, rotatePosition(rotatePosition(rotatePosition(position)))) != 0b00) {
            return true
        }

        if (cornerAt(s, position) == 0b11 && cornerAt(sup, above(position)) == 0b11) {
            return true
        }

        return false
    }
    function rotatePosition(p: number): number {
        return (p + 1) % 4 + ((p / 4) >>> 0) * 4
    }
    function below(p: number): number {
        if (p < 4) {
            return p
        }
        return p - 4
    }
    function above(p: number): number {
        if (p >= 3 * 4) {
            return p
        }
        return p + 4
    }



    function firstGroup(s: Shape): Shape {
        for (let i = 0; i < 16; i++) {
            if (cornerAt(s, i) != 0b00) {
                const group = connectedGroup(s, i, 0)

                const groupFilled = toFilled(group)
                const leftOver = toFilled(s) & ~groupFilled

                if (((groupFilled >>> 4) & leftOver) == 0 && ((groupFilled >>> 8) & leftOver) == 0 && ((groupFilled >>> 12) & leftOver) == 0) {
                    return group
                }
            }
        }
        return 0
    }

    function connectedGroup(s: Shape, position: number, group: Shape): Shape {
        if (cornerAt(s, position) == 0b00) {
            return group
        }

        if (cornerAt(group, position) == 0b01 || cornerAt(group, position) == 0b11) {
            return group
        }

        if (cornerAt(s, position) == 0b10) {
            return setCornerAt(group, position, 0b10)
        }

        group = setCornerAt(group, position, cornerAt(s, position))

        if (cornerAt(s, rotatePosition(position)) == 0b01) {
            group = connectedGroup(s, rotatePosition(position), group)
        }
        if (cornerAt(s, rotatePosition(rotatePosition(rotatePosition(position)))) == 0b01) {
            group = connectedGroup(s, rotatePosition(rotatePosition(rotatePosition(position))), group)
        }
        return group
    }

    function collapse(s: Shape): Shape {
        const sup = supported(s)
        if (sup == s) {
            return s
        }

        let unsupported = s & ~sup;
        const crystals = unsupported & (unsupported >>> 16)
        unsupported &= (~(crystals | (crystals << 16))) >>> 0

        let result = sup;

        while (unsupported != 0) {
            const group = firstGroup(unsupported)
            unsupported = unsupported & ~group

            let valid = 3;
            for (let i = 0; i < 4; i++) {
                if (((toFilled(group) >>> (i * 4)) & toFilled(result)) != 0 || ((group >>> (i * 4) << (i * 4)) >>> 0) != group) {
                    valid = i - 1
                    break
                }
            }

            result = result | ((group & 0b1111_1111_1111_1111) >>> (valid * 4))
            result = result | ((group >>> (valid * 4)) & ~0b1111_1111_1111_1111)
        }

        return result
    }

    function destoryCrystalAt(s: Shape, index: number): Shape {
        if (cornerAt(s, index) != 0b11) {
            return s
        }

        let result = setCornerAt(s, index, 0b00)
        result = destoryCrystalAt(result, rotatePosition(index))
        result = destoryCrystalAt(result, rotatePosition(rotatePosition(rotatePosition(index))))

        if ((index / 4) >>> 0 != 0) {
            result = destoryCrystalAt(result, below(index))
        }
        if ((index / 4) >>> 0 != 3) {
            result = destoryCrystalAt(result, above(index))
        }

        return result
    }

    function stack(bottom: Shape, top: Shape): Shape {
        for (let i = 0; i < 16; i++) {
            top = destoryCrystalAt(top, i)
        }

        let mask: Shape = 0
        while (top != 0) {
            let group = firstGroup(top)
            top = top & ~group

            while (((toFilled(group) & 0b1111) == 0) && group != 0) {
                group >>>= 4
            }

            const filledGroup = toFilled(group);
            if ((filledGroup & mask) != 0 || ((filledGroup << 12) & toFilled(bottom)) != 0) {
                mask |= toFilled(group)
                continue
            }

            const filledS = toFilled(bottom);

            let valid = 0
            for (let i = 3; i >= 0; i--) {
                if (((filledGroup << (i * 4)) & filledS) != 0) {
                    valid = i + 1
                    break
                }
            }

            bottom = bottom | ((group << (valid * 4)) & 0b1111_1111_1111_1111)
            bottom = bottom | ((group & ~0b1111_1111_1111_1111) << (valid * 4))
        }

        return bottom
    }

    function rotate(s: Shape): Shape {
        return ((s & 0b1110_1110_1110_1110_1110_1110_1110_1110) >>> 1) | ((s & 0b0001_0001_0001_0001_0001_0001_0001_0001) << 3)
    }

    function minimal(s: Shape): Shape {
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

    function isMinimal(s: Shape): boolean {
        return s == minimal(s)
    }
    function crystals(s: Shape): Shape {
        s = s & (s >>> 16)
        return s | (s << 16)
    }

    function unstackBottom(s: Shape): [bottom: Shape, top: Shape] {
        const mask = 0b1111_0000_0000_0000_1111;
        return [s & mask, (s & ~mask) >>> 4];
    }

    function removeBottomEmpty(s: Shape): Shape {
        if (s == 0) {
            return 0
        }
        const [bottom, top] = unstackBottom(s)
        if (bottom == 0) {
            return removeBottomEmpty(top)
        }
        return s
    }


    function isStackTopWithoutCrystals(s: Shape): boolean {
        const [bottom, top] = unstack(s)
        const c = crystals(top)

        const cBelow = c >>> 4;
        if ((toFilled(s) & toFilled(cBelow)) != toFilled(cBelow)) {
            return false
        }

        return (top & ~c) != 0 && (stack(bottom | c, top & ~c) >>> 0) == (s >>> 0)
    }

    function makeRecipe(s: Shape): Recipe {
        if (hardcodedRecipes[s]) {
            return hardcodedRecipes[s]
        }

        if (toFilled(s) <= 0b1111) {
            return {
                shape: s,
                operation: "trivial",
            }
        }

        if (!hasCrystal(topLayer(s))) {
            const [bottom, top] = unstack(s)
            return {
                operation: "stack",
                shape: s,
                original1: bottom,
                original2: top >>> ((layerCount(s) - 1) * 4),
            }
        }

        if ((s & ~0b0011_0011_0011_0011_0011_0011_0011_0011) == 0) {
            return { shape: s, operation: "half" }
        }

        if (isLeftRightValid(s)) {
            const right = (s & 0b0011_0011_0011_0011_0011_0011_0011_0011)
            const left = (s & ~0b0011_0011_0011_0011_0011_0011_0011_0011)
            if (collapse(left) == left && collapse(right) == right) {
                return { shape: s, operation: "combine", original1: left, original2: right }
            }
        }

        if (isUpDownValid(s)) {
            //rotate then do combine via isLeftRightValid
            return { shape: s, operation: "rotate", original1: rotate(rotate(rotate(s))) }
        }

        if (!isMinimal(s)) {
            if (isMinimal(rotate(s)) || isMinimal(rotate(rotate(s))) || isMinimal(rotate(rotate(rotate(s))))) {
                return { shape: s, operation: "rotate", original1: rotate(rotate(rotate(s))) }
            } else {
                return { shape: s, operation: "mirror", original1: mirror(s) }
            }
        }

        if (isStackTopWithoutCrystals(s)) {
            let [bottom, top] = unstack(s)
            bottom |= crystals(top)
            return {
                operation: "stack",
                shape: s,
                original1: bottom,
                original2: removeBottomEmpty(top & ~bottom),
            }
        }

        const pin = hardcodedPins[s];
        if (pin) {
            return { shape: s, operation: "pushPins", original1: pin }
        }

        const stack = hardcodedStacks[s];
        if (stack) {
            return { shape: s, operation: "stack", original1: stack, original2: removeBottomEmpty(s & ~stack) }
        }

        return { shape: s, operation: "unknown" }
    }
}
