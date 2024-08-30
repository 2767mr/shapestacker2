import { Shape } from "./shape";

const hardcodedPins = fetch('./hardcoded-pins.json').then(r => r.json());
const hardcodedStacks = fetch('./hardcoded-stacks.json').then(r => r.json());
const hardcodedRecipes = fetch('./hardcoded-halfs.json')
    .then(r => r.json())
    .then((recipes: Record<string, {
        operation: string,
        original1: number,
        original2: number,
    }>) =>
        Object.fromEntries(Object.entries(recipes)
            .map(([shape, recipe]) => [shape, { shape: +shape, ...recipe }]))
    );

export class Recipe {

    private constructor(
        public readonly shape: Shape,
        public readonly operation: string,
        public readonly original1: Shape = new Shape(0),
        public readonly original2: Shape = new Shape(0),
    ) { }

    public static async from(s: Shape): Promise<Recipe> {
        if (s.layerCount() <= 1) {
            return new Recipe(s, "trivial")
        }

        const hardCoded = (await hardcodedRecipes)[s.value];
        if (hardCoded) {
            return new Recipe(s, hardCoded.operation, new Shape(hardCoded.original1 ?? 0), new Shape(hardCoded.original2 ?? 0))
        }

        const pin = (await hardcodedPins)[s.value];
        if (pin) {
            return new Recipe(s, "pushPins", new Shape(pin));
        }

        const stack = (await hardcodedStacks)[s.value];
        if (stack) {
            return new Recipe(s, "stack", new Shape(stack), new Shape(s.value & ~stack).removeBottomEmpty());
        }

        if (!s.hasCrystalOnTopLayer()) {
            const [bottom, top] = s.unstack()
            return new Recipe(s, "stack", bottom, top.removeBottomEmpty());
        }

        // if (s.unsafeLeft().layerCount() == 0) {
        //     throw new Error("This should not happen");
        //     return new Recipe(s, "half");
        // }

        if (s.isLeftRightValid()) {
            return new Recipe(s, "combine", s.unsafeLeft(), s.unsafeRight());
        }

        if (s.isUpDownValid()) {
            //rotate then do combine via isLeftRightValid
            return new Recipe(s, "rotate", s.rotate270());
        }

        if (!s.isMinimal()) {
            if (s.rotate90().isMinimal() || s.rotate180().isMinimal() || s.rotate270().isMinimal()) {
                return new Recipe(s, "rotate", s.rotate270());
            } else {
                return new Recipe(s, "mirror", s.mirror());
            }
        }

        const [bottom, top] = s.unstackWithoutCrystals();
        return new Recipe(s, "stack", bottom, top);
    }
}