<script lang="ts">
	// Test helper: runs a callback during component initialisation, where getContext works.
	const props: { run: () => void } = $props();
	const init = () => props.run();
	init();
</script>
