import { mdsvex } from 'mdsvex';
import adapterNode from '@sveltejs/adapter-node';
import adapterAuto from '@sveltejs/adapter-auto';

// Vercel otomatis set VERCEL=1 saat build — pakai adapter-auto agar terdeteksi.
// Docker/lokal tidak punya env ini — pakai adapter-node (output ke build/).
const adapter = process.env.VERCEL ? adapterAuto() : adapterNode();

/** @type {import('@sveltejs/kit').Config} */
const config = {
	compilerOptions: {
		runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true)
	},
	kit: {
		adapter
	},
	preprocess: [mdsvex({ extensions: ['.svx', '.md'] })],
	extensions: ['.svelte', '.svx', '.md']
};

export default config;
