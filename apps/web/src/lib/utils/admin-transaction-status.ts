export const bookingStatusStyles = {
	pending_payment: { label: 'Pending payment', tone: 'bg-caution/10 text-caution' },
	confirmed: { label: 'Confirmed', tone: 'bg-positive/10 text-positive' },
	expired: { label: 'Expired', tone: 'bg-panel-muted text-ink-muted' },
	payment_review: { label: 'Payment review', tone: 'bg-information/10 text-information' }
};

export const paymentStatusStyles = {
	pending: { label: 'Pending', tone: 'bg-caution/10 text-caution' },
	paid: { label: 'Paid', tone: 'bg-positive/10 text-positive' },
	expired: { label: 'Expired', tone: 'bg-panel-muted text-ink-muted' },
	failed: { label: 'Failed', tone: 'bg-negative/10 text-negative' }
};
