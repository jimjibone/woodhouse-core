<script lang="ts">
	export let checked: boolean = false;
</script>

<button on:click class="button tag is-info is-light">
	{#if checked}
	<div class="iconWrapper">
		<svg
			class="checkIcon"
			viewBox="0 0 10 7"
			fill="none"
			xmlns="http://www.w3.org/2000/svg"
		>
			<path
				d="M4 4.586L1.707 2.293A1 1 0 1 0 .293 3.707l3 3a.997.997 0 0 0 1.414 0l5-5A1 1 0 1 0 8.293.293L4 4.586z"
				fill="currentColor"
				fill-rule="evenodd"
				clip-rule="evenodd"
			/>
		</svg>
	</div>
	{/if}
	<slot></slot>
</button>

<style>
	.iconWrapper {
		width: 20px;
		max-width: 20px;
		height: 12px;
		display: inline-block;
		vertical-align: middle;
		overflow-x: hidden;
		overflow-y: hidden;
		color: #296fa8;
	}

	.checkIcon {
		width: 12px;
		height: 10.909090909090908px;
		display: block;
	}
</style>
