import { Shape } from "./shape";

export function render(shape: Shape, target: HTMLElement) {
    target.innerHTML = renderString(shape);
}

function renderString(shape: Shape): string {
    let result = '';
    let value = shape.value;
    let code = shape.code ?? '';

    if (value == 0) {
        return result;
    }

    result += renderCorner(corner(value), code.slice(6), 0, 1);
    value = nextCorner(value);
    result += renderCorner(corner(value), code.slice(4), 270, 1);
    value = nextCorner(value);
    result += renderCorner(corner(value), code.slice(2), 180, 1);
    value = nextCorner(value);
    result += renderCorner(corner(value), code.slice(0), 90, 1);
    value = nextCorner(value);

    if (value == 0) {
        return result;
    }
    code = code.slice(9);

    result += renderCorner(corner(value), code.slice(6), 0, 0.8);
    value = nextCorner(value);
    result += renderCorner(corner(value), code.slice(4), 270, 0.8);
    value = nextCorner(value);
    result += renderCorner(corner(value), code.slice(2), 180, 0.8);
    value = nextCorner(value);
    result += renderCorner(corner(value), code.slice(0), 90, 0.8);
    value = nextCorner(value);

    if (value == 0) {
        return result;
    }
    code = code.slice(9);

    result += renderCorner(corner(value), code.slice(6), 0, 0.6);
    value = nextCorner(value);
    result += renderCorner(corner(value), code.slice(4), 270, 0.6);
    value = nextCorner(value);
    result += renderCorner(corner(value), code.slice(2), 180, 0.6);
    value = nextCorner(value);
    result += renderCorner(corner(value), code.slice(0), 90, 0.6);
    value = nextCorner(value);
    if (value == 0) {
        return result;
    }
    code = code.slice(9);

    result += renderCorner(corner(value), code.slice(6), 0, 0.4);
    value = nextCorner(value);
    result += renderCorner(corner(value), code.slice(4), 270, 0.4);
    value = nextCorner(value);
    result += renderCorner(corner(value), code.slice(2), 180, 0.4);
    value = nextCorner(value);
    result += renderCorner(corner(value), code.slice(0), 90, 0.4);
    value = nextCorner(value);

    return result;
}

function corner(value: number) {
    return (value & 0b1) | ((value >>> 15) & 0b10);
}
function nextCorner(value: number) {
    return (value & ~0b1_0000_0000_0000_0001) >>> 1;
}

function renderCorner(value: number, code: string, rotate: number, scale: number): string {
    let color = 'gray';
    switch (code[1]) {
        case 'r':
            color = 'red';
            break;
        case 'g':
            color = 'green';
            break;
        case 'b':
            color = 'blue';
            break;
        case 'y':
            color = 'yellow';
            break;
        case 'm':
            color = 'magenta';
            break;
        case 'c':
            color = 'cyan';
            break;
        case 'w':
            color = 'white';
            break;
    }

    switch (value) {
        case 0b00:
            return '';
        case 0b01: {
            switch (code[0]) {
                case 'R':
                    return drawRectangle(rotate, scale, color);
                case 'C':
                    return drawCircle(rotate, scale, color);
                case 'S':
                    return drawS(rotate, scale, color);
                case 'W':
                    return drawW(rotate, scale, color);
                default:
                    return drawCircle(rotate, scale, color);
            }
        }
        case 0b10:
            return drawPin(rotate, scale);
        case 0b11:
            return drawCrystal(rotate, scale, color)
        default:
            throw new Error('Invalid corner value')
    }
}

function rotateAndScale(rotate: number, scale: number) {
    return `transform="translate(50, 50) rotate(${rotate}) scale(${scale}, ${scale}) translate(-50, -50)"`
}

function drawCrystal(rotate: number, scale: number, color: string) {
    return `<path fill="${color}" stroke="#000" stroke-width="2" stroke-dasharray="5,5" d="M 50 50 L 0 50 a 50 50 90 0 1 50 -50 Z" ${rotateAndScale(rotate, scale)} />`
}

function drawCircle(rotate: number, scale: number, color: string) {
    return `<path fill="${color}" stroke="#000" stroke-width="2" d="M 50 50 L 1 50 a 49 49 90 0 1 49 -49 Z" ${rotateAndScale(rotate, scale)} />`
}

function drawRectangle(rotate: number, scale: number, color: string) {
    return `<path fill="${color}" stroke="#000" stroke-width="2" d="M 50 50 L 1 50 L 1 1 L 50 1 Z" ${rotateAndScale(rotate, scale)} />`
}

function drawS(rotate: number, scale: number, color: string) {
    return `<path fill="${color}" stroke="#000" stroke-width="2" d="M 50 50 L 50 25 L 0 0 L 25 50 Z" ${rotateAndScale(rotate, scale)} />`
}

function drawW(rotate: number, scale: number, color: string) {
    return `<path fill="${color}" stroke="#000" stroke-width="2" d="M 50 50 L 0 50 a 50 50 90 0 1 50 50 Z" ${rotateAndScale(rotate + 90, scale)} />`
}

function drawPin(rotate: number, scale: number) {
    return `<circle fill="#000" stroke="#333" stroke-width="2"  cx="30" cy="30" r="10" ${rotateAndScale(rotate, scale)} />`
}
