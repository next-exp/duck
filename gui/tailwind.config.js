/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {},
  },
  daisyui: {
    themes: ["light"],
  },
  plugins: [
	  require("daisyui"),
	  require('@headlessui/tailwindcss')({ prefix: 'ui' }),
  ],
}

