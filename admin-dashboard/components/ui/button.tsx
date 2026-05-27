'use client';

import * as React from 'react';
import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils/cn';

const buttonVariants = cva(
	'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 focus-visible:ring-offset-white disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-100 dark:focus-visible:ring-blue-400 dark:focus-visible:ring-offset-[#111827] disabled:border-gray-300 disabled:bg-gray-100 disabled:text-gray-600 dark:disabled:border-[#374151] dark:disabled:bg-[#1f2937] dark:disabled:text-gray-300',
	{
		variants: {
			variant: {
				default: 'bg-gray-900 text-white hover:bg-gray-800 dark:bg-[#2563eb] dark:text-white dark:hover:bg-blue-500',
				destructive: 'bg-red-700 text-white hover:bg-red-600 dark:bg-red-700 dark:text-white dark:hover:bg-red-600',
				outline: 'border border-gray-300 bg-white text-gray-950 hover:bg-gray-100 dark:border-[#374151] dark:bg-[#111827] dark:text-[#f9fafb] dark:hover:bg-[#1f2937]',
				secondary: 'bg-gray-100 text-gray-950 hover:bg-gray-200 dark:bg-[#1f2937] dark:text-[#f9fafb] dark:hover:bg-gray-700',
				ghost: 'text-gray-950 hover:bg-gray-100 dark:text-[#f9fafb] dark:hover:bg-[#1f2937]',
				link: 'text-gray-950 underline-offset-4 hover:underline dark:text-[#f9fafb]',
			},
			size: {
				default: 'h-9 px-4 py-2',
				sm: 'h-8 rounded-md px-3 text-xs',
				lg: 'h-10 rounded-md px-6',
				icon: 'h-9 w-9',
			},
		},
		defaultVariants: { variant: 'default', size: 'default' },
	},
);

export interface ButtonProps
	extends React.ButtonHTMLAttributes<HTMLButtonElement>,
		VariantProps<typeof buttonVariants> {
	asChild?: boolean;
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
	({ className, variant, size, asChild = false, ...props }, ref) => {
		const Comp = asChild ? Slot : 'button';
		return <Comp className={cn(buttonVariants({ variant, size, className }))} ref={ref} {...props} />;
	},
);
Button.displayName = 'Button';

export { buttonVariants };
