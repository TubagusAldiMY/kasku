<script lang="ts">
	import { onMount } from 'svelte';
	import { fade, fly } from 'svelte/transition';
	import { resolve } from '$app/paths';

	let { children } = $props();

	let currentSlide = $state(0);
	const slideIndices = [0, 1] as const;
	const slides = [
		{
			title: 'Pusat Terpadu untuk Keputusan Finansial Cerdas',
			description:
				'KasKu memberikan wawasan mendalam dan pandangan 360° terhadap kondisi finansial Anda.',
			icon: 'M13 10V3L4 14h7v7l9-11h-7z'
		},
		{
			title: 'Pantau Semua Aset dalam Satu Genggaman',
			description:
				'Kelola rekening bank, investasi emas, hingga kripto secara real-time untuk kebebasan finansial.',
			icon: 'M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z'
		}
	];

	onMount(() => {
		const interval = setInterval(() => {
			currentSlide = (currentSlide + 1) % slides.length;
		}, 5000);
		return () => clearInterval(interval);
	});
</script>

<div
	class="grid min-h-screen w-full grid-cols-1 bg-white font-sans selection:bg-teal-100 lg:grid-cols-2"
>
	<div
		class="relative flex flex-col items-center justify-center border-r border-gray-100 px-6 py-12 sm:px-16 lg:px-20"
	>
		<div class="absolute top-10 left-10 flex items-center gap-3">
			<div
				class="flex h-10 w-10 items-center justify-center rounded-xl bg-[#217b84] shadow-lg shadow-teal-900/20"
			>
				<svg class="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2.5"
						d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			</div>
			<span class="text-2xl font-bold tracking-tight text-[#0a2e31]">KasKu</span>
		</div>

		<div class="w-full max-w-[400px]">
			{@render children()}
		</div>

		<div
			class="absolute bottom-8 left-0 flex w-full flex-col justify-between gap-4 px-10 text-[11px] text-gray-400 sm:flex-row"
		>
			<p>© 2026 KasKu. All Rights Reserved.</p>
			<div class="flex gap-4 font-semibold tracking-widest text-gray-500 uppercase">
				<a href={resolve('/terms')} class="transition-colors hover:text-teal-600">Syarat</a>
				<a href={resolve('/privacy')} class="transition-colors hover:text-teal-600">Privasi</a>
			</div>
		</div>
	</div>

	<div class="relative hidden items-center justify-center overflow-hidden bg-[#0a2e31] lg:flex">
		<div class="absolute inset-0 bg-gradient-to-br from-[#1a5b63] via-[#0d3b41] to-[#0a2e31]"></div>
		<div
			class="absolute inset-0 opacity-10"
			style="background-image: url('data:image/svg+xml,%3Csvg width=%2240%22 height=%2240%22 viewBox=%220 0 40 40%22 xmlns=%22http://www.w3.org/2000/svg%22%3E%3Cpath d=%22M0 0h1v1H0z%22 fill=%22%23fff%22 fill-opacity=%22.4%22/%3E%3C/svg%3E');"
		></div>

		<div class="relative z-10 w-full max-w-lg px-12">
			<div class="relative mb-12 h-64">
				<div
					class="absolute top-0 left-0 w-full max-w-[320px] -rotate-2 transform rounded-3xl border border-white/10 bg-white/10 p-7 shadow-2xl backdrop-blur-xl transition-transform hover:rotate-0"
				>
					<div class="mb-1 text-[10px] font-bold tracking-[0.2em] text-teal-400 uppercase">
						Total Net Worth
					</div>
					<div class="text-3xl font-bold tracking-tight text-white">Rp 128.450.000</div>
					<div class="mt-5 flex items-center gap-3">
						<div class="h-1.5 flex-1 overflow-hidden rounded-full bg-white/10">
							<div
								class="h-full w-2/3 rounded-full bg-teal-400 shadow-[0_0_15px_rgba(45,212,191,0.6)]"
							></div>
						</div>
						<span class="text-xs font-bold text-white/60">64%</span>
					</div>
				</div>

				<div
					class="absolute right-0 -bottom-4 w-full max-w-[240px] rotate-3 transform rounded-2xl border border-white/10 bg-white/15 p-5 shadow-2xl backdrop-blur-xl transition-transform hover:rotate-0"
				>
					<div class="flex items-center gap-4">
						<div
							class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-teal-400/20 text-teal-400"
						>
							<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"
								><path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2"
								/></svg
							>
						</div>
						<div>
							<div class="text-[9px] font-bold tracking-widest text-teal-300 uppercase">
								Investment
							</div>
							<div class="text-lg font-bold text-white">Emas Antam</div>
						</div>
					</div>
				</div>
			</div>

			<div class="space-y-6 text-center">
				<div
					class="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl border border-white/10 bg-white/5 shadow-xl"
				>
					{#each slides as slide, i (i)}
						{#if i === currentSlide}
							<div in:fade={{ duration: 300 }} class="absolute">
								<svg
									class="h-8 w-8 text-teal-400"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d={slide.icon}
									/>
								</svg>
							</div>
						{/if}
					{/each}
				</div>

				<div class="relative h-32">
					{#each slides as slide, i (i)}
						{#if i === currentSlide}
							<div
								in:fly={{ y: 15, duration: 600 }}
								out:fade={{ duration: 300 }}
								class="absolute inset-0"
							>
								<h2 class="mb-3 text-3xl font-bold tracking-tight text-white">{slide.title}</h2>
								<p class="text-base leading-relaxed text-teal-50/60">{slide.description}</p>
							</div>
						{/if}
					{/each}
				</div>

				<div class="mt-6 flex justify-center gap-2">
					{#each slideIndices as i (i)}
						<button
							type="button"
							aria-label={`Tampilkan slide ${i + 1}`}
							onclick={() => (currentSlide = i)}
							class="h-1.5 rounded-full transition-all duration-300 {i === currentSlide
								? 'w-8 bg-teal-400'
								: 'w-4 bg-white/20'}"
						></button>
					{/each}
				</div>
			</div>
		</div>
	</div>
</div>

<style>
	:global(body) {
		margin: 0;
		padding: 0;
		overflow-x: hidden;
	}
</style>
