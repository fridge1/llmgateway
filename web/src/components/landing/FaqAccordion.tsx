import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { FAQ_ITEMS } from "./faq-data";

const FaqAccordion = () => {
  return (
    <section id="faq" className="py-20 bg-muted/30">
      <div className="max-w-3xl mx-auto px-6">
        <div className="text-center mb-10">
          <h2 className="text-3xl md:text-4xl font-bold text-foreground tracking-tight mb-3">
            常见问题
          </h2>
          <p className="text-muted-foreground">
            没找到答案？欢迎扫码联系客服
          </p>
        </div>

        <Accordion type="single" collapsible className="bg-card border border-border rounded-xl divide-y divide-border">
          {FAQ_ITEMS.map((item, i) => (
            <AccordionItem key={i} value={`item-${i}`} className="border-b-0 px-6 first:rounded-t-xl last:rounded-b-xl">
              <AccordionTrigger className="text-left font-semibold text-foreground hover:no-underline py-5">
                {item.q}
              </AccordionTrigger>
              <AccordionContent className="text-sm text-muted-foreground leading-relaxed pb-5">
                {item.a}
              </AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
      </div>
    </section>
  );
};

export default FaqAccordion;
