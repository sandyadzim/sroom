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
		pickupTime: pickupTime ? undefined : 'Choose a pickup time.',
		returnDate: !returnDate
			? 'Choose a return date.'
			: returnDate < pickupDate
				? 'Return date must be on or after pickup.'
				: undefined,
		returnTime: !returnTime
			? 'Choose a return time.'
			: returnDate === pickupDate && !!pickupTime && returnTime <= pickupTime
				? 'Return time must be after pickup.'
				: undefined
	};
}
