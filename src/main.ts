import { Recipe } from "./recipe";
import { render } from "./renderer";
import { Shape } from "./shape";

async function main() {
    Object.assign(globalThis, { solve });

    // solve('----CgCb:----Cy--:----P-cr:----Cwcm')
    solve('--cu--Cu:CucuCucu:--cucu--:----cu--')


    // printRecipe(parseShape('----CuCu:----Cu--:----P-cr:----Cucr'), 0)
    // console.log(printRecipe(mirror(parseShape('----CuCu:----Cu--:----P-cr:----Cucr')), 0))
}

if (location.hostname == 'localhost') {
    new EventSource('/esbuild').addEventListener('change', e => { location.reload() });
}

main()
async function solve(shape: string) {
    const drawQueue = [] as Shape[]

    async function printRecipe(shape: Shape, level: number, mirrored = false): Promise<string> {
        let result = '';
        const recipe = await Recipe.from(shape);
        if (recipe.operation == 'mirror') {
            return printRecipe(shape.mirror(), level, !mirrored)
        }

        const operation = mirrored && recipe.operation == 'right' ? 'left' : recipe.operation;

        function printShape(s: Shape, color: string) {
            s = mirrored ? s.mirror() : s;
            drawQueue.push(s);
            return `<span style="color:${color}">${s}</span> <svg class="shape-${s.toString().replaceAll(':', '_').replaceAll('c', 'A')}" viewBox="0 0 100 100" style="width: 64px; height: 64px" xmlns="http://www.w3.org/2000/svg"></svg>`;
        }

        result += '<div style="display: flex; flex-direction: row; place-items: center; gap: 1em; margin-top: 4px">' + '&nbsp;&nbsp;&nbsp;&nbsp;'.repeat(level);

        if (recipe.original1.value && recipe.original2.value) {
            result += (printShape(shape, 'black') + ' = <span style="color:blue">' + operation + '</span> ' + printShape(recipe.original1, 'magenta') + ' + ' + printShape(recipe.original2, 'red'));
            result += '</div>';

            if (recipe.original1.layerCount() > 1 && recipe.original2.layerCount() > 1) {
                level++
            }

            if (recipe.original1.layerCount() > 1) {
                result += await printRecipe(recipe.original1, level)
            }
            if (recipe.original2.layerCount() > 1) {
                result += await printRecipe(recipe.original2, level)
            }
        } else if (recipe.original1.value) {
            result += (printShape(shape, 'black') + ' = <span style="color:blue">' + operation + '</span> ' + printShape(recipe.original1, 'magenta'));
            result += '</div>';
            if (recipe.original1.layerCount() > 1) {
                result += await printRecipe(recipe.original1, level)
            }
        } else {
            result += (printShape(shape, 'black') + ' = <span style="color:blue">' + operation) + '</span>';
            result += '</div>';
        }

        return result
    }

    const output = document.getElementById('output') as HTMLDivElement;
    output.innerHTML = await printRecipe(new Shape(shape), 0);


    for (const shape of drawQueue) {
        for (const target of document.getElementsByClassName('shape-' + shape.toString().replaceAll(':', '_').replaceAll('c', 'A'))) {
            render(shape, target as HTMLElement)
        }
    }
    drawQueue.splice(0, drawQueue.length);


    render(new Shape(shape), document.getElementById('svg') as HTMLElement)
};
async function load(file: string): Promise<any> {
    if ('process' in globalThis) {
        const fs = globalThis['require']('node:fs') as typeof import('fs');
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

async function readSection(file: string, start: number, end: number): Promise<Uint8Array> {
    const fs = globalThis['require']('node:fs') as typeof import('fs');
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
