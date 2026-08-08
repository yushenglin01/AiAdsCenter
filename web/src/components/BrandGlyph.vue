<script setup lang="ts">
withDefaults(defineProps<{ label?: string }>(), { label: 'AdNova' })
</script>

<template>
  <svg class="brand-glyph" viewBox="0 0 48 48" role="img" :aria-label="label">
    <defs>
      <linearGradient id="adnova-orbit" x1="4" y1="6" x2="44" y2="42" gradientUnits="userSpaceOnUse">
        <stop stop-color="#72E6F2" />
        <stop offset="0.52" stop-color="#786BFF" />
        <stop offset="1" stop-color="#FF8B78" />
      </linearGradient>
    </defs>
    <path d="M8.8 27.8C5.9 17.6 12.5 8 23.5 8c9.4 0 16.2 6.4 16.2 15.1 0 9.8-8.7 17.4-19.5 16.7" fill="none" stroke="url(#adnova-orbit)" stroke-width="3.2" stroke-linecap="round" />
    <path d="M13.2 32.5 23.9 14l10.8 18.5M18 26.5h11.8" fill="none" stroke="currentColor" stroke-width="3.1" stroke-linecap="round" stroke-linejoin="round" />
    <circle cx="9.2" cy="28" r="3.2" fill="#72E6F2" />
    <circle cx="40" cy="22.8" r="2.2" fill="#FF8B78" />
  </svg>
</template>
