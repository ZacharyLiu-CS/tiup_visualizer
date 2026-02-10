<template>
  <svg class="connection-lines" :style="{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', pointerEvents: 'none', zIndex: 1 }">
    <line 
      v-for="(line, index) in lines" 
      :key="index"
      :x1="line.x1" 
      :y1="line.y1" 
      :x2="line.x2" 
      :y2="line.y2"
      :stroke="line.color"
      stroke-width="2"
      stroke-dasharray="5,5"
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
