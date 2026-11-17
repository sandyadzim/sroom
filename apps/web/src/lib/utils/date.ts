const witaDateTime = new Intl.DateTimeFormat('en-GB', {
	dateStyle: 'medium',
	timeStyle: 'short',
	timeZone: 'Asia/Makassar'
});

export function formatWitaDateTime(timestamp: string) {
	return witaDateTime.format(new Date(timestamp));
}

const witaDate = new Intl.DateTimeFormat('en-GB', {
	dateStyle: 'long',
	timeZone: 'Asia/Makassar'
});

export function formatWitaDate(timestamp: string) {
	return witaDate.format(new Date(timestamp));
}
