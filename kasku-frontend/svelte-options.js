// SvelteKit 3 tidak lagi membaca svelte.config.js; objek ini dipakai oleh sveltekit(...) di
// vite.config.ts dan oleh eslint.config.js (eslint-plugin-svelte), jadi satu sumber kebenaran.
import { mdsvex } from 'mdsvex';
import adapterNode from '@sveltejs/adapter-node';
import adapterAuto from '@sveltejs/adapter-auto';

// Vercel otomatis set VERCEL=1 saat build — pakai adapter-auto agar terdeteksi.
// Docker/lokal tidak punya env ini — pakai adapter-node (output ke build/).
const adapter = process.env.VERCEL ? adapterAuto() : adapterNode();

/** @type {import('@sveltejs/kit/vite').Config} */
const config = {
	compilerOptions: {
		runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true)
	},
	adapter,
	// Kit 3 menghapus alias bawaan $lib (digantikan #lib). Alias ini mempertahankan import lama;
	// ponytail: migrasi ke #lib (package.json "imports") bila alias $lib dihapus kit.
	alias: { $lib: 'src/lib' },
	preprocess: [mdsvex({ extensions: ['.svx', '.md'] })],
	extensions: ['.svelte', '.svx', '.md']
};

export default config;
