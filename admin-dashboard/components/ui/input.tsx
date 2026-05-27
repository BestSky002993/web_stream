'use client';

import * as React from 'react';
import { cn } from '@/lib/utils/cn';
import { formControlClass, formControlInteractiveClass, formDisabledClass, formFocusRingClass } from './form-styles';

export type InputProps = React.InputHTMLAttributes<HTMLInputElement>;

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
	({ className, type, ...props }, ref) => {
		return (
			<input
				type={type}
				ref={ref}
				className={cn(
					'flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-sm',
					'file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-current',
					formControlClass,
					formControlInteractiveClass,
					formFocusRingClass,
					formDisabledClass,
					className,
				)}
				{...props}
			/>
		);
	},
);
Input.displayName = 'Input';

export type TextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;

export const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
	({ className, ...props }, ref) => (
		<textarea
			ref={ref}
			className={cn(
				'flex min-h-[80px] w-full rounded-md border px-3 py-2 text-sm shadow-sm',
				formControlClass,
				formControlInteractiveClass,
				formFocusRingClass,
				formDisabledClass,
				className,
			)}
			{...props}
		/>
	),
);
Textarea.displayName = 'Textarea';
