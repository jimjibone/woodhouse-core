<script lang="ts">
	import { page } from "$app/state";
	import { goto } from '$app/navigation';
	import { loggedIn } from "$lib/stores/auth-store";
	import LoginForm from "$lib/components/login-form.svelte";

	$effect(() => {
		if ($loggedIn) {
			// Redirect to the prior page, or fallback to home.
			const redirectTo = page.url.searchParams.get('redirect');
			goto(redirectTo || '/');
		}
	})
</script>

<div class="bg-background flex min-h-svh flex-col items-center justify-center gap-6 p-6 md:p-10">
	<div class="w-full max-w-sm">
		<LoginForm />
	</div>
</div>
