<template>
  <div class="service-docs">
    <iframe class="docs-frame" :src="docsUrl" title="本站服务文档" />
  </div>
</template>

<script setup>
const docsUrl = '/docs'
</script>

<style scoped>
.service-docs {
  height: calc(100vh - 100px);
}
.docs-frame {
  width: 100%;
  height: 100%;
  border: none;
  background: #fff;
}
</style>
