export default function About() {
  return (
    <section className="landing-about">
      <span className="landing-about-tag">What is Botty</span>
      <p className="landing-about-lead">
        Botty is a WhatsApp bot that helps you track tasks all the way to done, even when the group chat is a mess.
      </p>
      <p className="landing-about-body">
        Someone reports a problem in the group. You tag Botty on it and it shows up as a task on your dashboard.
        When it's fixed, you say so in the chat and the task closes. If five people report the same thing, Botty puts it all under one task. Complaints don't get lost in the chaos and every action maintains an audit log.
      </p>
    </section>
  );
}
