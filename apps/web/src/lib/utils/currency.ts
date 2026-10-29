const idr = new Intl.NumberFormat('id-ID', {
	style: 'currency',
	currency: 'IDR',
	currencyDisplay: 'code',
	maximumFractionDigits: 0
});

export const formatIDR = (amount: number) => idr.format(amount);
