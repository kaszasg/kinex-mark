<script lang="ts">
	/** Measuring rule. Divides a section or a timeline, once or twice per page. */
	let { class: className = '' }: { class?: string } = $props();
</script>

<span class="kx-ticks {className}" aria-hidden="true"></span>
