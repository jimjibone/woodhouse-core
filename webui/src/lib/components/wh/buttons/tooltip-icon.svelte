<script lang="ts" module>
	import { type VariantProps, tv } from "tailwind-variants";
	const iconVariants = tv({
		// base: "focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium outline-none transition-all focus-visible:ring-[3px] disabled:pointer-events-none disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:opacity-50 [&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0 size-9",
		base: "inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium outline-none transition-all [&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0 size-9",
		variants: {
			variant: {
				default: "hover:bg-accent hover:text-accent-foreground dark:hover:bg-accent/50",
				destructive: "bg-destructive shadow-xs hover:bg-destructive/90 focus-visible:ring-destructive/20 dark:focus-visible:ring-destructive/40 dark:bg-destructive/60 text-white",
			}
		},
		defaultVariants: {
			variant: "default",
		}
	});

	export type IconVariant = VariantProps<typeof iconVariants>["variant"];
</script>

<script lang="ts">
	import * as Tooltip from "$lib/components/ui/tooltip";
	import { type Snippet } from "svelte";

	let {
		variant = "default",
		disabled = false,
		tooltip,
		children,
	} : {
		variant: IconVariant,
		disabled?: boolean,
		tooltip: string,
		children: Snippet
	} = $props();

	// const className = cn(
	// 	"focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium outline-none transition-all focus-visible:ring-[3px] disabled:pointer-events-none disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:opacity-50 [&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0",
	// 	"size-5"
	// )
</script>

<Tooltip.Provider>
	<Tooltip.Root ignoreNonKeyboardFocus={true}>
		<Tooltip.Trigger class={iconVariants({ variant })} {disabled}>
			{@render children()}
		</Tooltip.Trigger>
		<Tooltip.Content>
			<p>{tooltip}</p>
		</Tooltip.Content>
	</Tooltip.Root>
</Tooltip.Provider>
