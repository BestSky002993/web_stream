export const menuSurfaceClass =
	'z-[100] border border-[#374151] bg-[#111827] text-[#f9fafb] shadow-2xl shadow-black/50 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-400';

export const menuItemClass =
	'border-b border-[#374151] text-[#f9fafb] transition-colors last:border-b-0 hover:bg-[#1f2937] hover:text-white focus:bg-[#1f2937] focus:text-white focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-blue-400 data-[highlighted]:bg-[#1f2937] data-[highlighted]:text-white data-[highlighted]:ring-1 data-[highlighted]:ring-inset data-[highlighted]:ring-gray-500 data-[state=checked]:bg-[#2563eb] data-[state=checked]:text-white data-[state=checked]:font-medium data-[highlighted]:data-[state=checked]:bg-blue-500 data-[highlighted]:data-[state=checked]:ring-blue-200';

export const menuDestructiveItemClass =
	'border-b border-[#374151] text-red-200 transition-colors last:border-b-0 hover:bg-red-900 hover:text-white focus:bg-red-900 focus:text-white focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-red-300 data-[highlighted]:bg-red-900 data-[highlighted]:text-white data-[highlighted]:ring-1 data-[highlighted]:ring-inset data-[highlighted]:ring-red-400';

export const menuSeparatorClass = 'bg-[#374151]';

export const menuMutedTextClass = 'text-gray-300';

export const menuFocusRingClass =
	'focus:outline-none focus:ring-2 focus:ring-blue-400 focus:ring-offset-2 focus:ring-offset-[#111827]';

export const commandListClass = 'max-h-80 overflow-y-auto p-1';

export const commandItemClass = menuItemClass;

export const popoverSurfaceClass = menuSurfaceClass;
