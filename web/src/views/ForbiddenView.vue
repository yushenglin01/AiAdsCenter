<script setup lang="ts">
import AppShell from '@/components/AppShell.vue'
</script>
<template><AppShell><el-result icon="warning" title="无权限访问" sub-title="当前账号缺少访问此页面所需的角色。"><template #extra><el-button @click="$router.push('/')">返回总览</el-button></template></el-result></AppShell></template>
