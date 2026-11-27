// Each scenario is a script: messages appear in order (delay = ms after the
// previous one). `bot` messages are Botty's replies. `reply` renders a quoted
// parent. `pan` (optional) slides the window over to the dashboard afterwards.
// TODO: change — all names, group titles and task numbers below are stubs.

const BASE_TASKS = [
  { ref: 'BOT-12', title: 'Projector not working in hall B', status: 'open' },
  { ref: 'BOT-9', title: 'Wifi down on floor 2', status: 'done' },
];

export const SCENARIOS = {
  link: {
    group: 'Hostel Maintenance',
    pre: 'generate', // starts on the Groups page: click "Generate link code"
    pan: { kind: 'groups', group: { name: 'Hostel Maintenance', members: 4, allowed: 1 } },
    messages: [
      { id: 'a', from: 'Rahul', text: '@botty link BOTYY-XXXX', delay: 900 },
      { id: 'b', bot: true, replyTo: 'a', text: 'yayaa yipee linked', delay: 1300 },
    ],
  },

  create: {
    group: 'Hostel Maintenance',
    messages: [
      { id: 'a', from: 'Asha', text: '@botty the whater is overflowing in room 35 :(', delay: 900 },
      { id: 'b', bot: true, replyTo: 'a', text: 'done', delay: 1200 },
    ],
    pan: { tasks: BASE_TASKS, added: { ref: 'BOT-13', title: 'Water overflowing in room 35', status: 'open', reports: 1 } },
  },

  close: {
    group: 'Hostel Maintenance',
    messages: [
      { id: 'a', from: 'Asha', text: '@botty the water is overflowing in room 35 :(', delay: 800 },
      { id: 'b', from: 'Dev', text: '@botty looking into it', replyTo: 'a', delay: 1400 },
      { id: 'c', bot: true, replyTo: 'b', text: 'done', delay: 1000 },
      { id: 'd', from: 'Dev', text: 'sent someone to ur room @botty', replyTo: 'a', delay: 2200 },
      { id: 'e', bot: true, replyTo: 'd', text: 'done', delay: 1000 },
    ],
  },

  dupes: {
    group: 'Hostel Maintenance',
    messages: [
      { id: 'a', from: 'Asha', text: 'my fan isnt working', delay: 800 },
      { id: 'b', from: 'Dev', text: '@botty looking into it', replyTo: 'a', delay: 1400 },
      { id: 'c', bot: true, replyTo: 'b', text: 'done', delay: 1000 },
      { id: 'd', from: 'Kabir', text: '@botty, mine too', replyTo: 'a', delay: 1500 },
      { id: 'e', bot: true, replyTo: 'd', text: 'done', delay: 1000 },
    ],
    pan: { tasks: BASE_TASKS, added: { ref: 'BOT-13', title: 'Fan not working', status: 'open', reports: 2 } },
  },

  hinglish: {
    group: 'Hostel Maintenance',
    messages: [
      { id: 'a', from: 'Meera', image: 'washer', text: '@botty washing machine kaam nhi kar rha', delay: 900 },
      { id: 'b', bot: true, replyTo: 'a', text: 'done', delay: 1200 },
    ],
    pan: { tasks: BASE_TASKS, added: { ref: 'BOT-13', title: 'Washing machine not working', status: 'open', reports: 1, image: 'washer' } },
  },

  guard: {
    group: 'Hostel Maintenance',
    messages: [
      { id: 'a', from: 'Dev', text: '@botty anyone up for chai?', delay: 900 },
      { id: 'b', bot: true, replyTo: 'a', text: 'thats not a task', delay: 1200 },
      { id: 'c', from: '+91 98•••• ••21', text: '@botty the AC is broken', delay: 1800 },
      { id: 'd', bot: true, replyTo: 'c', text: 'only allowed users are allowed to tag me', delay: 1200 },
    ],
  },
};
