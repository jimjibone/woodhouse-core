export function makeAcronym(fullName?: string | null, username?: string): string {
	const source = fullName?.trim() || username?.trim() || '';

	if (!source) return '';

	// Normalize whitespace
	const parts = source.split(/\s+/);

	if (parts.length > 1) {
		// Full name → take first letter of first + last word
		const first = parts[0][0];
		const last = parts[parts.length - 1][0];
		return (first + last).toUpperCase();
	}

	// Single word: use first alphabetical character
	const match = parts[0].match(/[A-Za-z]/);
	if (match) return match[0].toUpperCase();

	return ''; // fallback if no valid character
}
