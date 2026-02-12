<template>
  <svg class="connection-lines" :style="{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', pointerEvents: 'none', zIndex: 5 }">
    <path 
      v-for="(line, index) in lines" 
      :key="index"
      :d="line.path"
      :stroke="line.color"
      stroke-width="2.5"
      fill="none"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
</template>

<script>
export default {
  name: 'ConnectionLines',
  props: {
    lines: {
      type: Array,
      default: () => []
    }
  }
}
</script>
