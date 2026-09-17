<script lang="ts">
	import { onMount } from 'svelte';
	import { WoodhouseIcon } from '$lib/components/wh/icons';
	import { buttonVariants } from '$lib/components/ui/button';
	import * as Collapsible from '$lib/components/ui/collapsible/index.js';
	import { DownloadIcon, ChevronDownIcon } from '@lucide/svelte';
	import { cn } from '$lib/utils';

	type TrustInfo = {
		caSubject: string;
		caFingerprintSha256: string;
		caNotAfter: string;
		dnsNames: string[];
		leafNotAfter: string;
	};

	type PlatformKey = 'ios' | 'mac' | 'android' | 'windows' | 'linux';

	// iPadOS Safari reports itself as Macintosh when running in desktop mode,
	// so a touch-capable "Mac" is treated as an iPad instead.
	function detectPlatform(): PlatformKey | null {
		const ua = navigator.userAgent;
		if (/iPhone|iPad/.test(ua)) return 'ios';
		if (/Macintosh/.test(ua)) return navigator.maxTouchPoints > 1 ? 'ios' : 'mac';
		if (/Android/.test(ua)) return 'android';
		if (/Windows/.test(ua)) return 'windows';
		if (/Linux/.test(ua)) return 'linux';
		return null;
	}

	let info = $state<TrustInfo | null>(null);
	let loadError = $state<string | null>(null);
	let loading = $state(true);

	// Each section tracks its own open state, so opening one does not close
	// another. Only the section matching the visiting device starts open.
	let iosOpen = $state(false);
	let macOpen = $state(false);
	let androidOpen = $state(false);
	let windowsOpen = $state(false);
	let linuxOpen = $state(false);
	let firefoxOpen = $state(false);

	onMount(async () => {
		switch (detectPlatform()) {
			case 'ios':
				iosOpen = true;
				break;
			case 'mac':
				macOpen = true;
				break;
			case 'android':
				androidOpen = true;
				break;
			case 'windows':
				windowsOpen = true;
				break;
			case 'linux':
				linuxOpen = true;
				break;
		}
		try {
			const res = await fetch('/api/trust/info');
			if (!res.ok) throw new Error('status ' + res.status);
			info = await res.json();
		} catch {
			loadError = 'Could not load certificate details.';
		} finally {
			loading = false;
		}
	});

	// Matches the app's own port so the linked address works whether the
	// server is reached via a proxy or its default port.
	const port = $derived(typeof window !== 'undefined' && window.location.port ? ':' + window.location.port : '');
</script>

{#snippet trigger(title: string, open: boolean)}
	<Collapsible.Trigger
		class="flex w-full items-center gap-1.5 text-sm font-medium hover:text-foreground transition-colors cursor-pointer select-none"
	>
		<ChevronDownIcon class={cn('size-4 transition-transform duration-200', open && 'rotate-180')} />
		{title}
	</Collapsible.Trigger>
{/snippet}

<div class="bg-background flex min-h-svh flex-col items-center justify-center gap-6 p-6 md:p-10">
	<div class="w-full max-w-lg flex flex-col gap-6">
		<div class="flex flex-col items-center gap-2 text-center">
			<div class="flex size-12 items-center justify-center rounded-md">
				<WoodhouseIcon class="size-10" />
			</div>
			<h1 class="text-xl font-bold">Trust this server</h1>
			<p class="text-muted-foreground text-sm">
				This server uses its own local certificate authority to sign its HTTPS certificate. Installing that authority
				on this device makes the browser trust this server without warnings. Installation is per device and only
				affects trust for this server's certificate authority.
			</p>
		</div>

		<a href="/api/trust/ca.crt" download="woodhouse-ca.crt" class={cn(buttonVariants(), 'w-full')}>
			<DownloadIcon />
			Download certificate authority
		</a>

		<div class="flex flex-col gap-3 rounded-xl border bg-card/50 p-4 shadow-sm">
			{#if loading}
				<p class="text-muted-foreground text-sm">Loading certificate details...</p>
			{:else if loadError}
				<p class="text-muted-foreground text-sm">{loadError}</p>
			{:else if info}
				<div class="grid gap-1">
					<span class="text-sm font-medium">{info.caSubject}</span>
					<span class="text-muted-foreground text-xs">
						Compare this fingerprint with what the device shows before trusting it:
					</span>
					<span class="font-mono text-xs break-all">{info.caFingerprintSha256}</span>
				</div>

				<div class="grid gap-1">
					<span class="text-sm font-medium">Valid addresses</span>
					<ul class="grid gap-0.5">
						{#each info.dnsNames as name (name)}
							<li>
								<a href={`https://${name}${port}`} class="text-sm underline underline-offset-4">
									{`https://${name}${port}`}
								</a>
							</li>
						{/each}
					</ul>
				</div>

				<p class="text-muted-foreground text-xs">
					The server certificate renews itself automatically (current one expires {new Date(
						info.leafNotAfter
					).toLocaleDateString()}); the CA does not need reinstalling.
				</p>
			{/if}
		</div>

		<div class="flex flex-col gap-2">
			<Collapsible.Root bind:open={iosOpen}>
				{@render trigger('iPhone / iPad', iosOpen)}
				<Collapsible.Content class="pt-2 pl-5.5">
					<ol class="list-decimal list-inside text-sm text-muted-foreground grid gap-1">
						<li>Open this page in Safari and tap Download. Allow the profile when prompted.</li>
						<li>
							Go to Settings &gt; Profile Downloaded (or Settings &gt; General &gt; VPN &amp; Device Management) &gt;
							Install.
						</li>
						<li>
							Go to Settings &gt; General &gt; About &gt; Certificate Trust Settings and enable full trust for
							"Woodhouse Local CA".
						</li>
						<li>Reopen this site.</li>
					</ol>
				</Collapsible.Content>
			</Collapsible.Root>

			<Collapsible.Root bind:open={macOpen}>
				{@render trigger('Mac', macOpen)}
				<Collapsible.Content class="pt-2 pl-5.5">
					<ol class="list-decimal list-inside text-sm text-muted-foreground grid gap-1">
						<li>Download the certificate, then open the file. It lands in Keychain Access under login.</li>
						<li>
							Double-click "Woodhouse Local CA", expand Trust, and set "When using this certificate" to Always
							Trust.
						</li>
						<li>Restart the browser.</li>
					</ol>
					<p class="text-muted-foreground text-sm pt-1">
						Firefox uses its own certificate store, not the Keychain - see the Firefox section below.
					</p>
				</Collapsible.Content>
			</Collapsible.Root>

			<Collapsible.Root bind:open={androidOpen}>
				{@render trigger('Android', androidOpen)}
				<Collapsible.Content class="pt-2 pl-5.5">
					<ol class="list-decimal list-inside text-sm text-muted-foreground grid gap-1">
						<li>Download the certificate.</li>
						<li>
							Go to Settings &gt; Security &amp; privacy &gt; More security settings &gt; Encryption &amp;
							credentials &gt; Install a certificate &gt; CA certificate. The exact path varies by vendor.
						</li>
						<li>Pick the downloaded file and confirm.</li>
						<li>Reopen this site.</li>
					</ol>
				</Collapsible.Content>
			</Collapsible.Root>

			<Collapsible.Root bind:open={windowsOpen}>
				{@render trigger('Windows', windowsOpen)}
				<Collapsible.Content class="pt-2 pl-5.5">
					<ol class="list-decimal list-inside text-sm text-muted-foreground grid gap-1">
						<li>Download the certificate, open the file, then click Install Certificate.</li>
						<li>
							Choose Current User, then "Place all certificates in the following store" and pick Trusted Root
							Certification Authorities.
						</li>
						<li>Restart the browser.</li>
					</ol>
				</Collapsible.Content>
			</Collapsible.Root>

			<Collapsible.Root bind:open={linuxOpen}>
				{@render trigger('Linux', linuxOpen)}
				<Collapsible.Content class="pt-2 pl-5.5">
					<ol class="list-decimal list-inside text-sm text-muted-foreground grid gap-1">
						<li>
							Copy the file to /usr/local/share/ca-certificates/woodhouse-ca.crt (Debian/Ubuntu) and run
							<code class="font-mono text-xs">sudo update-ca-certificates</code>, or on Fedora copy it to
							/etc/pki/ca-trust/source/anchors/ and run
							<code class="font-mono text-xs">sudo update-ca-trust</code>.
						</li>
						<li>
							Chrome/Chromium use their own NSS database on Linux: import it under Settings &gt; Privacy and
							security &gt; Security &gt; Manage certificates &gt; Authorities, or run
							<code class="font-mono text-xs"
								>certutil -d sql:$HOME/.pki/nssdb -A -t "C,," -n "Woodhouse Local CA" -i woodhouse-ca.crt</code
							>.
						</li>
					</ol>
				</Collapsible.Content>
			</Collapsible.Root>

			<Collapsible.Root bind:open={firefoxOpen}>
				{@render trigger('Firefox (all platforms)', firefoxOpen)}
				<Collapsible.Content class="pt-2 pl-5.5">
					<ol class="list-decimal list-inside text-sm text-muted-foreground grid gap-1">
						<li>
							Go to Settings &gt; Privacy &amp; Security &gt; Certificates &gt; View Certificates &gt; Authorities
							&gt; Import, and tick "Trust this CA to identify websites".
						</li>
						<li>
							Alternatively, set <code class="font-mono text-xs">security.enterprise_roots.enabled</code> to true in
							about:config to use the OS trust store instead.
						</li>
					</ol>
				</Collapsible.Content>
			</Collapsible.Root>
		</div>

		<a href="/login" class="text-muted-foreground text-center text-sm underline underline-offset-4">Back to login</a>
	</div>
</div>
