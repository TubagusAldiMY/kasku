<script lang="ts">
	import { resolve } from '$app/paths';

	let { children } = $props();
</script>

<div class="flex min-h-screen items-center justify-center bg-paper px-4 py-8 sm:px-6">
	<div
		class="grid w-full max-w-[1040px] overflow-hidden rounded-2xl border border-ink/10 bg-card shadow-xl lg:min-h-[580px] lg:grid-cols-[44fr_56fr]"
	>
		<!-- Brand panel (mockup: deep-green split, desktop only) -->
		<div class="relative hidden flex-col justify-between bg-ink p-12 text-card lg:flex">
			<span class="font-serif text-[26px] leading-none tracking-tight"
				>Kas<em class="text-mint">Ku</em></span
			>

			<div class="max-w-sm">
				<p class="font-serif text-[30px] leading-tight tracking-tight xl:text-[34px]">
					“Kamu tidak perlu jadi kaya untuk mulai tertib. Kamu perlu tertib untuk mulai kaya.”
				</p>
				<p class="mt-6 text-sm text-card/55">Catatanmu aman, offline maupun online.</p>
			</div>

			<p class="text-xs text-card/40">© 2026 KasKu</p>
		</div>

		<!-- Form column -->
		<div class="flex flex-col px-8 py-10 sm:px-12">
			<a
				href={resolve('/')}
				class="inline-block font-serif text-2xl leading-none tracking-tight text-ink lg:hidden"
			>
				Kas<em class="text-teal">Ku</em>
			</a>

			<div class="flex flex-1 items-center justify-center py-8">
				<div class="w-full max-w-[360px]">
					{@render children()}
				</div>
			</div>

			<footer class="flex items-center justify-center gap-2 text-[11px] text-ink/40">
				<span class="lg:hidden">© 2026 KasKu ·</span>
				<a href={resolve('/terms')} class="transition-colors hover:text-teal">Ketentuan</a>
				<span aria-hidden="true">·</span>
				<a href={resolve('/privacy')} class="transition-colors hover:text-teal">Privasi</a>
			</footer>
		</div>
	</div>
</div>

<style>
	:global(body) {
		overflow-x: hidden;
	}
</style>
