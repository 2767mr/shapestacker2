enum CornerType {
    NONE,
    FILLED,
    PIN,
    CRYSTAL,
}

export class Shape {
    public readonly value: number;
    public readonly code?: string;
    constructor(
        value: number | string
    ) {
        if (typeof value === 'string') {
            this.code = value;
            this.value = Shape.from(value).value;
        } else {
            this.value = value >>> 0;
        }
    }

    private static from(text: string): Shape {
        let result = new Shape(0);

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
                        result = result.setCornerAt((3 - index % 4) + ((index / 4) >>> 0) * 4, CornerType.FILLED)
                        break;
                    case 'P':
                        result = result.setCornerAt((3 - index % 4) + ((index / 4) >>> 0) * 4, CornerType.PIN)
                        break;
                    case 'c':
                        result = result.setCornerAt((3 - index % 4) + ((index / 4) >>> 0) * 4, CornerType.CRYSTAL)
                        break;
                }
                index++
            }
        }
        return new Shape(result.value >>> 0)
    }

    get [Symbol.toStringTag]() {
        return "Shape";
    }

    private toFilled(): Shape {
        return new Shape((this.value | (this.value >>> 16)) & 0b1111_1111_1111_1111);
    }

    public layerCount(): number {
        return 4 - (((Math.clz32(this.toFilled().value) - 16) / 4) >>> 0)
    }

    public toString() {
        if (this.code) {
            return this.code;
        }

        const layers = this.layerCount();

        let result = ''

        result += this.cornerString(3)
        result += this.cornerString(2)
        result += this.cornerString(1)
        result += this.cornerString(0)

        if (layers <= 1) {
            return result
        }

        result += ':'
        result += this.cornerString(7)
        result += this.cornerString(6)
        result += this.cornerString(5)
        result += this.cornerString(4)

        if (layers == 2) {
            return result
        }

        result += ':'
        result += this.cornerString(11)
        result += this.cornerString(10)
        result += this.cornerString(9)
        result += this.cornerString(8)

        if (layers == 3) {
            return result
        }

        result += ':'
        result += this.cornerString(15)
        result += this.cornerString(14)
        result += this.cornerString(13)
        result += this.cornerString(12)

        return result
    }


    private cornerString(corner: number): string {
        switch (this.cornerAt(corner)) {
            case CornerType.FILLED:
                return 'Cu';
            case CornerType.PIN:
                return 'P-';
            case CornerType.CRYSTAL:
                return 'cu';
            default:
                return '--';
        }
    }

    private setCornerAt(index: number, value: CornerType): Shape {
        return new Shape(this.value & ~((0b1 << index) >>> 0) & ~(1 << (index + 16)) | ((value & 1) << index) >>> 0 | ((value & 0b10) << (index + 15)) >>> 0)
    }

    private cornerAt(index: number): CornerType {
        return ((this.value >>> index) & 1) | ((this.value >>> (index + 15)) & 0b10)
    }

    public hasCrystalOnTopLayer(): boolean {
        let b = this.toFilled().value;
        b |= (b & 0b0111_0111_0111_0111) << 1;
        b |= (b & 0b0011_0011_0011_0011) << 2;

        const topLayer = (this.value >>> ((3 - ((Math.clz32(b) - 16) / 4) >>> 0) * 4)) & 0b1111_0000_0000_0000_1111;
        return ((topLayer >>> 16) & topLayer) !== 0;
    }

    public unstack(): [bottom: Shape, top: Shape] {
        if (this.value == 0) {
            return [new Shape(0), new Shape(0)]
        }
        const mask = (0b1111_0000_0000_0000_1111 << ((this.layerCount() - 1) * 4)) >>> 0
        return [new Shape(this.value & ~mask), new Shape(this.value & mask)]
    }

    public removeBottomEmpty(): Shape {
        if (this.value == 0) {
            return new Shape(0);
        }

        const mask = 0b1111_0000_0000_0000_1111;
        let value = this.value;
        while (value !== 0 && (value & mask) === 0) {
            value >>>= 4;
        }

        return new Shape(value);
    }

    /** If called without proper checking this will result in an invalid shape */
    public unsafeRight(): Shape {
        return new Shape(this.value & 0b0011_0011_0011_0011_0011_0011_0011_0011)
    }

    /** If called without proper checking this will result in an invalid shape */
    public unsafeLeft(): Shape {
        return new Shape(this.value & ~0b0011_0011_0011_0011_0011_0011_0011_0011)
    }

    /** If called without proper checking this will result in an invalid shape */
    public unsafeUp(): Shape {
        return new Shape(this.value & 0b1001_1001_1001_1001_1001_1001_1001_1001)
    }
    /** If called without proper checking this will result in an invalid shape */
    public unsafeDown(): Shape {
        return new Shape(this.value & ~0b1001_1001_1001_1001_1001_1001_1001_1001)
    }

    public isLeftRightValid(): boolean {
        const right = this.unsafeRight()
        const left = this.unsafeLeft()
        return left.collapse().value == left.value && right.collapse().value == right.value
    }

    public isUpDownValid(): boolean {
        const up = this.unsafeUp()
        const down = this.unsafeDown()
        return up.collapse().value == up.value && down.collapse().value == down.value
    }

    public rotate90(): Shape {
        return new Shape(((this.value & 0b1110_1110_1110_1110_1110_1110_1110_1110) >>> 1) | ((this.value & 0b0001_0001_0001_0001_0001_0001_0001_0001) << 3))
    }

    public rotate180(): Shape {
        return this.rotate90().rotate90();
    }

    public rotate270(): Shape {
        return this.rotate180().rotate90();
    }

    private collapse(): Shape {
        const sup = this.supported()
        if (sup.value == this.value) {
            return this;
        }

        let unsupported = this.value & ~sup;
        const crystals = unsupported & (unsupported >>> 16)
        unsupported &= (~(crystals | (crystals << 16))) >>> 0

        let result = sup;

        while (unsupported != 0) {
            const group = new Shape(unsupported).firstGroup()
            unsupported = unsupported & ~group.value

            let valid = 3;
            for (let i = 0; i < 4; i++) {
                if (((group.toFilled().value >>> (i * 4)) & result.toFilled().value) != 0 || ((group.value >>> (i * 4) << (i * 4)) >>> 0) != group.value) {
                    valid = i - 1
                    break
                }
            }

            result = new Shape(result.value | ((group.value & 0b1111_1111_1111_1111) >>> (valid * 4))
                | ((group.value >>> (valid * 4)) & ~0b1111_1111_1111_1111));
        }

        return result
    }

    private supported(): Shape {
        let sup = new Shape(0)
        let newSupported = new Shape(this.value & 0b1111_0000_0000_0000_1111);

        while (newSupported.value !== sup.value) {
            sup = newSupported
            for (let i = 0; i < 16; i++) {
                if (this.isSupported(this, i, sup)) {
                    newSupported = newSupported.setCornerAt(i, this.cornerAt(i))
                }
            }
        }

        return sup
    }
    private isSupported(s: Shape, position: number, sup: Shape): boolean {
        if (s.cornerAt(position) === 0b00) {
            return false
        }

        if (((position / 4) >>> 0) == 0) {
            return true
        }

        if (sup.cornerAt(position) !== 0b00) {
            return true
        }

        if (sup.cornerAt(this.below(position)) != 0b00) {
            return true
        }

        if (s.cornerAt(position) != 0b10 && s.cornerAt(this.rotatePosition(position)) != 0b10 && sup.cornerAt(this.rotatePosition(position)) != 0b00) {
            return true
        }

        if (s.cornerAt(position) != 0b10 && s.cornerAt(this.rotatePosition(this.rotatePosition(this.rotatePosition(position)))) != 0b10 && sup.cornerAt(this.rotatePosition(this.rotatePosition(this.rotatePosition(position)))) != 0b00) {
            return true
        }

        if (s.cornerAt(position) == 0b11 && sup.cornerAt(this.above(position)) == 0b11) {
            return true
        }

        return false
    }
    private rotatePosition(p: number): number {
        return (p + 1) % 4 + ((p / 4) >>> 0) * 4
    }
    private below(p: number): number {
        if (p < 4) {
            return p
        }
        return p - 4
    }
    private above(p: number): number {
        if (p >= 3 * 4) {
            return p
        }
        return p + 4
    }

    private firstGroup(): Shape {
        for (let i = 0; i < 16; i++) {
            if (this.cornerAt(i) != 0b00) {
                const group = this.connectedGroup(i, new Shape(0));

                const groupFilled = group.toFilled().value;
                const leftOver = this.toFilled().value & ~groupFilled;

                if (((groupFilled >>> 4) & leftOver) == 0 && ((groupFilled >>> 8) & leftOver) == 0 && ((groupFilled >>> 12) & leftOver) == 0) {
                    return group
                }
            }
        }
        return new Shape(0);
    }

    private connectedGroup(position: number, group: Shape): Shape {
        if (this.cornerAt(position) == 0b00) {
            return group
        }

        if (group.cornerAt(position) == 0b01 || group.cornerAt(position) == 0b11) {
            return group
        }

        if (this.cornerAt(position) == 0b10) {
            return group.setCornerAt(position, 0b10)
        }

        group = group.setCornerAt(position, this.cornerAt(position))

        if (this.cornerAt(this.rotatePosition(position)) == 0b01) {
            group = this.connectedGroup(this.rotatePosition(position), group)
        }
        if (this.cornerAt(this.rotatePosition(this.rotatePosition(this.rotatePosition(position)))) == 0b01) {
            group = this.connectedGroup(this.rotatePosition(this.rotatePosition(this.rotatePosition(position))), group)
        }
        return group
    }


    public mirror(): Shape {
        const mask1 = 0x1111_1111
        const mask2 = 0x8888_8888
        const mask3 = mask1 | mask2

        const mask4 = 0x2222_2222
        const mask5 = 0x4444_4444
        const mask6 = mask4 | mask5

        let s = this.value
        s = (s & ~mask3) | ((s & mask1) << 3) | ((s & mask2) >>> 3)
        s = (s & ~mask6) | ((s & mask4) << 1) | ((s & mask5) >>> 1)
        return new Shape(s);
    }

    public isMinimal(): boolean {
        const s = this.value;
        return s === Math.min(
            s,
            this.rotate90().value,
            this.rotate180().value,
            this.rotate270().value,
            this.mirror().value,
            this.mirror().rotate90().value,
            this.mirror().rotate180().value,
            this.mirror().rotate270().value,
        );
    }

    public unstackWithoutCrystals(): [bottom: Shape, top: Shape] {
        let [bottom, top] = this.unstack();
        const bottomWithCrystals = new Shape(bottom.value | top.crystals().value);
        const topWithoutCrystals = new Shape(top.value & ~top.crystals().value);
        return [bottomWithCrystals, topWithoutCrystals];
    }

    private crystals(): Shape {
        let s = this.value;
        s = s & (s >>> 16)
        return new Shape(s | (s << 16));
    }

}