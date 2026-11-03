<!--
  LoginShowcase — left column of the login page: logo, pitch, servers illustration, topic chips.
  Static content, no props.
-->
<template>
  <section
    aria-label="Platforma haqida"
    class="flex flex-col gap-6 px-4 pb-2 pt-6 sm:px-8 sm:pt-10 lg:min-h-screen lg:flex-[1_1_520px] lg:gap-7 lg:px-14 lg:py-11"
  >
    <AppLogo to="/login" />

    <div class="flex max-w-[540px] flex-col gap-3.5 lg:mt-auto">
      <span class="font-mono text-[13px] text-accent">$ ./start-journey.sh</span>
      <h1 class="font-mono text-[28px] font-bold leading-[1.15] tracking-[-0.02em] sm:text-[38px]">
        Haqiqiy serverda o'rganing.
        <span class="text-accent">Xatodan qo'rqmang.</span>
      </h1>
      <p class="text-[15px] text-text-2">
        Har bir vazifa uchun alohida Linux konteyneri. Brauzerdagi terminalda ishlaysiz — platforma yechimingizni
        avtomatik tekshiradi.
      </p>
    </div>

    <img
      src="/assets/servers.svg"
      alt="Server stoykalari va terminal oynasi illyustratsiyasi"
      width="600"
      height="400"
      class="block h-auto w-full max-w-[520px]"
      decoding="async"
    />

    <ul class="flex list-none flex-wrap gap-2.5 font-mono text-meta lg:mb-auto" aria-label="Mavzular">
      <li
        v-for="topic in TOPICS"
        :key="topic.name"
        class="rounded-full border border-border-strong bg-[rgba(15,21,32,0.8)] px-2.5 py-1"
        :style="{ color: topic.color }"
      >
        {{ topic.name }}
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import AppLogo from '@/components/ui/AppLogo.vue'

const TOPICS = [
  { name: 'linux', color: '#3DDC97' },
  { name: 'docker', color: '#38BDF8' },
  { name: 'systemd', color: '#A78BFA' },
  { name: 'ufw', color: '#FB7185' },
  { name: 'cron', color: '#2DD4BF' },
  { name: 'nginx', color: '#F5B342' }
] as const
</script>
