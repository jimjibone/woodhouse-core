<script lang="ts">
	import type { HTMLAttributes } from "svelte/elements";
	import {
		FieldGroup,
		Field,
		FieldLabel,
		FieldDescription,
		FieldError,
	} from "$lib/components/ui/field/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import { Button } from "$lib/components/ui/button/index.js";
	import { cn, type WithElementRef } from "$lib/utils.js";
	import { WoodhouseIcon } from "./wh/icons";
	import { doLogin, noAdminsRegistered, type RefreshResultType } from "@/stores/auth-store";
    import { toSentenceCase } from "@/tools/headline-case";

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> = $props();

	const id = $props.id();

	let loginError: RefreshResultType | null = $state(null);

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		const form = event.target as HTMLFormElement;
		const data = new FormData(form);
		const username = data.get(`username-${id}`);
		const password = data.get(`password-${id}`);

		if (username && password) {
			const res = await doLogin(username.toString(), password.toString());
			loginError = res;
		}
	}
</script>

<div class={cn("flex flex-col gap-6", className)} bind:this={ref} {...restProps}>
	<form onsubmit={handleSubmit}>
		<FieldGroup>
			<div class="flex flex-col items-center gap-2 text-center">
				<a href="##" class="flex flex-col items-center gap-2 font-medium">
					<div class="flex size-12 items-center justify-center rounded-md">
						<WoodhouseIcon class="size-10" />
					</div>
					<span class="sr-only">Woodhouse</span>
				</a>
				<h1 class="text-xl font-bold">Welcome to Woodhouse</h1>
				{#if $noAdminsRegistered}
					<FieldDescription>
						Please create your admin account below.
					</FieldDescription>
				{/if}
			</div>
			<Field>
				<FieldLabel for="username-{id}">Username</FieldLabel>
				<Input name="username-{id}" type="text" placeholder="" required />
			</Field>
			<Field>
				<FieldLabel for="password-{id}">Password</FieldLabel>
				<Input name="password-{id}" type="password" placeholder="" required />
			</Field>
			{#if loginError}
				<FieldError>{toSentenceCase(loginError.errorMsg)}</FieldError>
			{/if}
			<Field>
				<Button type="submit">{$noAdminsRegistered ? "Create User" : "Login"}</Button>
			</Field>
		</FieldGroup>
	</form>
	<!-- <FieldDescription class="px-6 text-center">
		By clicking continue, you agree to our <a href="##">Terms of Service</a> and
		<a href="##">Privacy Policy</a>.
	</FieldDescription> -->
</div>
