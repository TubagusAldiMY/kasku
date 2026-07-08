<script lang="ts">
	import { resolve } from '$app/paths';

	let { children } = $props();
</script>

<div class="grid min-h-screen w-full bg-card lg:grid-cols-[44fr_56fr]">
	<!-- Brand panel (mockup: deep-green split, desktop only) -->
	<div class="relative hidden flex-col justify-between bg-ink p-14 text-card lg:flex">
		<span class="font-serif text-[26px] leading-none tracking-tight"
			>Kas<em class="text-mint">Ku</em></span
		>

		<div class="max-w-md">
			<p class="font-serif text-[34px] leading-tight tracking-tight xl:text-[40px]">
				“Kamu tidak perlu jadi kaya untuk mulai tertib. Kamu perlu tertib untuk mulai kaya.”
			</p>
			<p class="mt-6 text-sm text-card/55">Catatanmu aman, offline maupun online.</p>
		</div>

		<p class="text-xs text-card/40">© 2026 KasKu</p>
	</div>

	<!-- Form column -->
	<div class="relative flex flex-col items-center justify-center px-6 py-14 sm:px-14">
		<a
			href={resolve('/')}
			class="mb-10 inline-block font-serif text-2xl leading-none tracking-tight text-ink lg:hidden"
		>
			Kas<em class="text-teal">Ku</em>
		</a>

		<div class="w-full max-w-[380px]">
			{@render children()}
		</div>

		<div
			class="absolute bottom-8 left-0 flex w-full flex-col justify-between gap-3 px-8 text-[11px] text-ink/40 sm:flex-row sm:px-14"
		>
			<p>© 2026 KasKu</p>
			<div class="flex gap-5 font-medium">
				<a href={resolve('/terms')} class="transition-colors hover:text-teal">Ketentuan</a>
				<a href={resolve('/privacy')} class="transition-colors hover:text-teal">Privasi</a>
			</div>
		</div>
	</div>
</div>

<style>
	:global(body) {
		overflow-x: hidden;
	}
</style>
