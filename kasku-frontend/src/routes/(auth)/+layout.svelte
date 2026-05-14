<script lang="ts">
    import { onMount } from 'svelte';
    import { fade, fly } from 'svelte/transition';

    let { children } = $props();

    let currentSlide = $state(0);
    const slides = [
       {
          title: 'Pusat Terpadu untuk Keputusan Finansial Cerdas',
          description: 'KasKu memberikan wawasan mendalam dan pandangan 360° terhadap kondisi finansial Anda.',
          icon: 'M13 10V3L4 14h7v7l9-11h-7z'
       },
       {
          title: 'Pantau Semua Aset dalam Satu Genggaman',
          description: 'Kelola rekening bank, investasi emas, hingga kripto secara real-time untuk kebebasan finansial.',
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

<div class="grid min-h-screen w-full grid-cols-1 lg:grid-cols-2 bg-white font-sans selection:bg-teal-100">

    <div class="relative flex flex-col items-center justify-center px-6 py-12 sm:px-16 lg:px-20 border-r border-gray-100">

       <div class="absolute top-10 left-10 flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-[#217b84] shadow-lg shadow-teal-900/20">
             <svg class="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
             </svg>
          </div>
          <span class="text-2xl font-bold tracking-tight text-[#0a2e31]">KasKu</span>
       </div>

       <div class="w-full max-w-[400px]">
          {@render children()}
       </div>

       <div class="absolute bottom-8 left-0 w-full px-10 flex flex-col sm:flex-row justify-between text-[11px] text-gray-400 gap-4">
          <p>© 2026 KasKu. All Rights Reserved.</p>
          <div class="flex gap-4 font-semibold uppercase tracking-widest text-gray-500">
             <a href="/terms" class="hover:text-teal-600 transition-colors">Syarat</a>
             <a href="/privacy" class="hover:text-teal-600 transition-colors">Privasi</a>
          </div>
       </div>
    </div>

    <div class="relative hidden overflow-hidden bg-[#0a2e31] lg:flex items-center justify-center">

       <div class="absolute inset-0 bg-gradient-to-br from-[#1a5b63] via-[#0d3b41] to-[#0a2e31]"></div>
       <div class="absolute inset-0 opacity-10" style="background-image: url('data:image/svg+xml,%3Csvg width=%2240%22 height=%2240%22 viewBox=%220 0 40 40%22 xmlns=%22http://www.w3.org/2000/svg%22%3E%3Cpath d=%22M0 0h1v1H0z%22 fill=%22%23fff%22 fill-opacity=%22.4%22/%3E%3C/svg%3E');"></div>

       <div class="relative z-10 w-full max-w-lg px-12">

          <div class="relative h-64 mb-12">
             <div class="absolute top-0 left-0 w-full max-w-[320px] rounded-3xl border border-white/10 bg-white/10 p-7 shadow-2xl backdrop-blur-xl -rotate-2 transform transition-transform hover:rotate-0">
                <div class="mb-1 text-[10px] font-bold uppercase tracking-[0.2em] text-teal-400">Total Net Worth</div>
                <div class="text-3xl font-bold text-white tracking-tight">Rp 128.450.000</div>
                <div class="mt-5 flex items-center gap-3">
                   <div class="h-1.5 flex-1 rounded-full bg-white/10 overflow-hidden">
                      <div class="h-full w-2/3 rounded-full bg-teal-400 shadow-[0_0_15px_rgba(45,212,191,0.6)]"></div>
                   </div>
                   <span class="text-xs font-bold text-white/60">64%</span>
                </div>
             </div>

             <div class="absolute -bottom-4 right-0 w-full max-w-[240px] rounded-2xl border border-white/10 bg-white/15 p-5 shadow-2xl backdrop-blur-xl rotate-3 transform transition-transform hover:rotate-0">
                <div class="flex items-center gap-4">
                   <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-teal-400/20 text-teal-400">
                      <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2" /></svg>
                   </div>
                   <div>
                      <div class="text-[9px] font-bold uppercase tracking-widest text-teal-300">Investment</div>
                      <div class="text-lg font-bold text-white">Emas Antam</div>
                   </div>
                </div>
             </div>
          </div>

          <div class="text-center space-y-6">
             <div class="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-white/5 border border-white/10 shadow-xl">
                {#each slides as slide, i}
                   {#if i === currentSlide}
                      <div in:fade={{ duration: 300 }} class="absolute">
                         <svg class="h-8 w-8 text-teal-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={slide.icon} />
                         </svg>
                      </div>
                   {/if}
                {/each}
             </div>

             <div class="relative h-32">
                {#each slides as slide, i}
                   {#if i === currentSlide}
                      <div in:fly={{ y: 15, duration: 600 }} out:fade={{ duration: 300 }} class="absolute inset-0">
                         <h2 class="text-3xl font-bold text-white mb-3 tracking-tight">{slide.title}</h2>
                         <p class="text-teal-50/60 leading-relaxed text-base">{slide.description}</p>
                      </div>
                   {/if}
                {/each}
             </div>

             <div class="flex justify-center gap-2 mt-6">
                {#each slides as _, i}
                   <button
                      type="button"
                      aria-label={`Tampilkan slide ${i + 1}`}
                      onclick={() => (currentSlide = i)}
                      class="h-1.5 transition-all duration-300 rounded-full {i === currentSlide ? 'w-8 bg-teal-400' : 'w-4 bg-white/20'}"
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
