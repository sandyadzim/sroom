const witaDateTime = new Intl.DateTimeFormat('en-GB', {
	dateStyle: 'medium',
	timeStyle: 'short',
	timeZone: 'Asia/Makassar'
});

export function formatWitaDateTime(timestamp: string) {
	return witaDateTime.format(new Date(timestamp));
}
