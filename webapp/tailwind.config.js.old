/** @type {import('tailwindcss').Config} */
const mode = process.env.NODE_ENV || 'development';
const prod = mode === 'production';
module.exports = {
	content: ["./src/**/*.{html,js,ts,svelte}"],
	darkMode: 'class',
	theme: {
		extend: {},
	},
	plugins: [],
	content: [
		'./public/**/*.html',
		"./src/**/*.svelte",
	],
	enabled: prod // disable purge in dev
}
