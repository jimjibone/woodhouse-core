<script lang="ts">
	import { clock } from '$lib/stores/clock';
	import { cn } from '$lib/utils';
    import { onDestroy } from 'svelte';

	let {
		past,
		class: className = ""
	} : {
		past: Date,
		class?: string
	} = $props();

	let time: number = $state(0);
	const unsub = clock.subscribe(v => time = v);
	onDestroy(unsub);

	let text: string = $state("none");
	let isWarning: boolean = $state(false);
	let isDanger: boolean = $state(false);
	$effect(() => {
		if (past.getTime() === 0) {
			text = `never`;
			isWarning = false;
			isDanger = false;
			return;
		}
		const seconds = Math.max(Math.floor((time - past.getTime()) / 1000), 0);
		if (seconds < 60) {
			text = `${seconds}s ago`;
			isWarning = false;
			isDanger = false;
			return;
		}
		if (seconds < 3600) {
			text = `${Math.floor(seconds / 60)}m ago`;
			isWarning = false;
			isDanger = false;
			return;
		}
		if (seconds < 86400) {
			text = `${Math.floor(seconds / 3600)}h ago`;
			isWarning = false;
			isDanger = false;
			return;
		}
		const days = Math.floor(seconds / 86400);
		if (days <= 7) {
			text = `${days}d ago`;
			isWarning = true;
			isDanger = false;
			return;
		}
		const weeks = Math.floor(days / 7);
		text = `${weeks}w ago`;
		isWarning = false;
		isDanger = true;
	});
</script>

<span class={cn("text-muted-foreground", className, isWarning && "text-warning-foreground", isDanger && "text-error-foreground")}>{text}</span>
