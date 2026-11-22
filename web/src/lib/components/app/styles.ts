// Class lists that bring the generated shadcn-svelte (nova) components to the GRAPHICS.md sizes.
// The generated files stay untouched (plan 8.1.5); callers add these classes instead.

/** Button sizes of GRAPHICS.md 10.3: default 36 px, sm 28 px, icon 36x36 (the nova defaults are smaller). */
export const buttonSize = {
	default: 'h-9 gap-2 px-4 text-[13px] leading-5',
	sm: 'h-7 gap-2 px-3 text-[13px] leading-5',
	lg: 'h-11 gap-2 px-6 text-[15px]',
	icon: 'size-9 pointer-coarse:size-9',
	iconSm: 'size-7 pointer-coarse:size-9'
} as const;

/**
 * The destructive button of GRAPHICS.md 10.3 (filled destructive background). Use it on a Button
 * with the default variant: the nova destructive variant is a tinted surface instead.
 */
export const destructiveButton =
	'bg-destructive text-destructive-foreground hover:bg-destructive focus-visible:ring-destructive/40';

/**
 * Invalid field of GRAPHICS.md 6.2 and 10.9: the destructive border only. nova adds a 3 px
 * translucent ring halo and halves the border in the dark theme (a dark-theme variant, which only
 * an important utility outranks). The focus ring stays the global 2 px solid ring.
 */
export const invalidField = 'aria-invalid:ring-0 aria-invalid:border-destructive!';

/**
 * Input fields of GRAPHICS.md 10.9: 36 px high, 13 px text, radius-md, transparent also in the
 * dark theme (nova fills fields with input/30 through a dark-theme variant, which only an important
 * utility outranks); mono when the value is machine data. Invalid: `invalidField`.
 */
export const inputSize = `h-9 rounded-md bg-transparent! px-3 text-[13px] ${invalidField}`;

/** Card surface of GRAPHICS.md 10.0: 1 px border instead of the nova ring, radius-lg, no shadow. */
export const cardSurface = 'gap-0 rounded-lg border py-0 ring-0 shadow-none';

/** Line tabs of GRAPHICS.md 10.18 for Tabs.List (use with variant="line", which clears the dark fills). */
export const lineTabsList =
	'h-auto w-full justify-start gap-6 rounded-none border-b bg-transparent p-0';

/** Line tabs of GRAPHICS.md 10.18 for Tabs.Trigger. */
export const lineTabsTrigger =
	'relative flex-none rounded-none border-0 bg-transparent px-0 pt-2 pb-2.5 text-[13px] leading-5 font-medium text-muted-foreground shadow-none hover:text-foreground data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none data-[state=active]:after:absolute data-[state=active]:after:inset-x-0 data-[state=active]:after:-bottom-px data-[state=active]:after:h-0.5 data-[state=active]:after:bg-primary';

/** Table header cell of GRAPHICS.md 10.7. */
export const tableHead =
	'h-auto px-4 py-2 font-mono text-[11px] leading-4 font-medium tracking-[0.08em] uppercase text-muted-foreground bg-muted';

/** Table body cell of GRAPHICS.md 10.7. */
export const tableCell = 'px-4 py-[9px] text-[13px] leading-5 whitespace-nowrap';

/**
 * Dialog box of GRAPHICS.md 10.8 for Dialog.Content and AlertDialog.Content; opens and closes in
 * 200 ms ease-out (6.1; nova: 100 ms). User text (a 500-character title, a URL, one long word)
 * never widens it: the single grid column may shrink to 0 (nova's implicit auto column grew to
 * the widest unbreakable line), its children may shrink, long words wrap anywhere, and a box
 * taller than the window scrolls, so the footer always stays reachable.
 */
export const dialogBox =
	'w-[min(440px,calc(100vw-32px))] max-w-[440px] grid-cols-[minmax(0,1fr)] *:min-w-0 wrap-anywhere max-h-[calc(100dvh-32px)] overflow-y-auto gap-4 rounded-lg border bg-popover p-6 text-popover-foreground shadow-md ring-0 duration-200 ease-out sm:max-w-[440px]';

/**
 * Dialog footer of GRAPHICS.md 10.8: buttons aligned right with space-2, no band. The nova footer
 * is a muted band sized for its p-4 box (-mx-4 -mb-4), so inside the p-6 dialogBox it stood about
 * 9 px in from the edges.
 */
export const dialogFooter =
	'mx-0 mb-0 flex-row justify-end gap-2 rounded-none border-t-0 bg-transparent p-0';

/**
 * radius-md (4 px) for toggle items, also on the outer corners of a segmented group (GRAPHICS.md
 * 5.1, 10.3). The generated item rounds them with rounded-lg (6 px) through these exact variant
 * chains; the same chains with -md replace them in the class merge.
 */
const toggleRadius = [
	'rounded-md',
	'group-data-horizontal/toggle-group:data-[spacing=0]:first:rounded-l-md',
	'group-data-horizontal/toggle-group:data-[spacing=0]:last:rounded-r-md',
	'group-data-vertical/toggle-group:data-[spacing=0]:first:rounded-t-md',
	'group-data-vertical/toggle-group:data-[spacing=0]:last:rounded-b-md'
].join(' ');

/**
 * ToggleGroup.Item states of GRAPHICS.md 6.2: the selected item gets the primary marker (a 2 px
 * primary line like the active tab), the accent surface and foreground text; the others stay
 * muted. The nova "on" fill alone (bg-muted) is about 1.2:1 against the page in the dark theme.
 */
export const toggleItem =
	'relative text-muted-foreground hover:text-foreground data-[state=on]:bg-accent data-[state=on]:text-foreground data-[state=on]:after:absolute data-[state=on]:after:inset-x-1 data-[state=on]:after:bottom-0 data-[state=on]:after:h-0.5 data-[state=on]:after:bg-primary ' +
	toggleRadius;

/**
 * Checkbox of GRAPHICS.md 10.5: 16 px box, 1 px input border, radius-sm, transparent while
 * unchecked also in the dark theme (nova fills it with input/30 through a dark-theme variant,
 * which only an important utility outranks), primary when checked; no ring halo when invalid.
 */
export const checkboxBox = 'rounded-sm data-unchecked:bg-transparent! aria-invalid:ring-0';

// Type styles of GRAPHICS.md 3.5 as utilities. The .heading-3/.body-sm/.kx-eyebrow classes of
// app.css live in @layer components, so on a generated component the nova utilities on the same
// element (text-base, leading-none, text-sm, text-xs) win; these lists replace them instead.

/** heading-3: 18/26, weight 500 (dialog, sheet and empty-state titles). */
export const typeHeading3 = 'font-sans text-lg leading-[26px] font-medium tracking-normal';

/** body-sm: 13/20, weight 400 (dialog descriptions, dense UI text). */
export const typeBodySm = 'font-sans text-[13px] leading-5 font-normal';

/** kx-eyebrow: mono 11/16, weight 500, 0.08em, upper case, muted-foreground. */
export const typeEyebrow =
	'font-mono text-[11px] leading-4 font-medium tracking-[0.08em] uppercase text-muted-foreground';

/**
 * Per-variant fixes for the generated Button (GRAPHICS.md 10.3, 6.2, 5.1): radius-md instead of
 * nova's rounded-lg, the primary hover shifts toward the foreground instead of fading, outline and
 * ghost hover on the accent surface, and the outline button stays transparent in the dark theme
 * (nova fills it with input/30 through a dark-theme variant, which only an important utility outranks).
 * The app's Button wrapper applies them; use them directly only on buttonVariants() links.
 */
export const buttonTone = {
	default:
		'rounded-md hover:bg-[color-mix(in_oklab,var(--primary)_88%,var(--foreground))] aria-expanded:bg-[color-mix(in_oklab,var(--primary)_88%,var(--foreground))]',
	outline: `rounded-md border-input bg-transparent! hover:bg-accent! hover:text-foreground aria-expanded:bg-accent! aria-expanded:text-foreground ${invalidField}`,
	secondary: 'rounded-md hover:bg-accent aria-expanded:bg-accent',
	ghost:
		'rounded-md hover:bg-accent! hover:text-foreground aria-expanded:bg-accent! aria-expanded:text-foreground',
	destructive: 'rounded-md',
	link: 'h-auto rounded-sm px-0 text-primary underline underline-offset-3'
} as const;

/** Tooltip surface of GRAPHICS.md 2.3/5.4: popover, 1 px border, radius-md, shadow-popover. */
export const tooltipSurface =
	'rounded-md border bg-popover px-3 py-2 text-popover-foreground shadow-md';
