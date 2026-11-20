const dayMs = 86_400_000;
const witaDate = new Intl.DateTimeFormat('sv-SE', {
	timeZone: 'Asia/Makassar',
	year: 'numeric',
	month: '2-digit',
	day: '2-digit'
});

export type Schedule = {
	pickupDate: string;
	pickupTime: string;
	returnDate: string;
	returnTime: string;
};

export function earliestPickupDate(now = new Date()) {
	return witaDate.format(new Date(now.getTime() + dayMs));
}

export function inclusiveRentalDays(pickupDate: string, returnDate: string) {
	return (Date.parse(returnDate) - Date.parse(pickupDate)) / dayMs + 1;
}

export function validateSchedule(schedule: Schedule, earliestPickup: string) {
	const { pickupDate, pickupTime, returnDate, returnTime } = schedule;
	return {
		pickupDate: !pickupDate
			? 'Choose a pickup date.'
			: pickupDate < earliestPickup
				? 'Pickup starts tomorrow in Samarinda.'
				: undefined,
		pickupTime: !pickupTime
			? 'Choose a pickup time.'
			: !isHandoverSlot(pickupTime)
				? 'Choose a 30-minute slot from 08:00 to 18:00 WITA.'
				: undefined,
		returnDate: !returnDate
			? 'Choose a return date.'
			: returnDate < pickupDate
				? 'Return date must be on or after pickup.'
				: undefined,
		returnTime: !returnTime
			? 'Choose a return time.'
			: !isHandoverSlot(returnTime)
				? 'Choose a 30-minute slot from 08:00 to 18:00 WITA.'
				: returnDate === pickupDate && !!pickupTime && returnTime <= pickupTime
					? 'Return time must be after pickup.'
					: undefined
	};
}

export function isHandoverSlot(time: string) {
	return /^(?:0[89]|1[0-7]):(?:00|30)$|^18:00$/.test(time);
}
