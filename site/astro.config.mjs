// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
	integrations: [
		starlight({
			title: 'hop.top/aps',
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/hop-top/aps' }],
			sidebar: [
				{
					label: 'Guides',
					items: [
						{ label: 'Getting Started', slug: 'guides/getting-started' },
						{ label: 'Profiles', slug: 'guides/profiles' },
						{ label: 'Isolation Levels', slug: 'guides/isolation' },
					],
				},
				{
					label: 'Reference',
					autogenerate: { directory: 'reference' },
				},
			],
		}),
	],
});
